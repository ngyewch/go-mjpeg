package mjpeg

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
)

// MultipartMixedReplaceHandler implements a http.Handler that serves multipart/x-mixed-replace content.
type MultipartMixedReplaceHandler struct {
	partContentType string
	onNewRequest    func() (<-chan []byte, <-chan error, func() error, error)
}

// NewMultipartMixedReplaceHandler creates a new MultipartMixedReplaceHandler.
func NewMultipartMixedReplaceHandler(partContentType string, onNewRequest func() (<-chan []byte, <-chan error, func() error, error)) *MultipartMixedReplaceHandler {
	return &MultipartMixedReplaceHandler{
		partContentType: partContentType,
		onNewRequest:    onNewRequest,
	}
}

func (handler *MultipartMixedReplaceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dataCh, errCh, closer, err := handler.onNewRequest()
	if err != nil {
		slog.Error("error handling request",
			slog.Any("err", err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if closer != nil {
		defer func() {
			err := closer()
			if err != nil {
				slog.Error("error closing request",
					slog.Any("err", err),
				)
			}
		}()
	}

	boundaryString := rand.Text()
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", boundaryString))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, pre-check=0, post-check=0, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	for {
		select {
		case <-r.Context().Done():
			return
		case err := <-errCh:
			slog.Error("subscription error",
				slog.Any("err", err),
			)
			return
		case frameBytes := <-dataCh:
			err := func() error {
				_, err = fmt.Fprintf(w, "--%s\n", boundaryString)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(w, "Content-Type: %s\n", handler.partContentType)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(w, "Content-Length: %d\n", len(frameBytes))
				if err != nil {
					return err
				}
				_, err = w.Write([]byte("\n"))
				if err != nil {
					return err
				}
				_, err = w.Write(frameBytes)
				if err != nil {
					return err
				}
				_, err = w.Write([]byte("\n"))
				if err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				slog.Error("error writing frame",
					slog.Any("err", err),
				)
			}
		}
	}
}
