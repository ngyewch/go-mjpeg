package mjpeg

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
)

// HTTPStreamDecoder is an HTTP multipart/x-mixed-replace MJPEG stream decoder.
type HTTPStreamDecoder struct {
	multipartReader *multipart.Reader
}

// NewHTTPStreamDecoder creates a new HTTPStreamDecoder.
func NewHTTPStreamDecoder(r io.Reader, boundary string) *HTTPStreamDecoder {
	multipartReader := multipart.NewReader(r, boundary)
	return &HTTPStreamDecoder{
		multipartReader: multipartReader,
	}
}

// NewHTTPStreamDecoderFromHTTPResponse creates a new HTTPStreamDecoder from a *http.Response.
func NewHTTPStreamDecoderFromHTTPResponse(httpResponse *http.Response) (*HTTPStreamDecoder, error) {
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", httpResponse.StatusCode)
	}
	contentType := httpResponse.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	if mediaType != "multipart/x-mixed-replace" {
		return nil, fmt.Errorf("expected multipart/x-mixed-replace, got %s", mediaType)
	}
	boundary, ok := params["boundary"]
	if !ok {
		return nil, fmt.Errorf("boundary parameter missing")
	}
	httpStreamDecoder := NewHTTPStreamDecoder(httpResponse.Body, boundary)
	return httpStreamDecoder, nil
}

// Next returns the next MJPEG frame as a byte array.
func (decoder *HTTPStreamDecoder) Next(ctx context.Context) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	part, err := decoder.multipartReader.NextPart()
	if err != nil {
		return nil, err
	}
	contentType := part.Header.Get("Content-Type")
	if contentType != "image/jpeg" {
		return nil, fmt.Errorf("expected image/jpeg content type, got %s", contentType)
	}
	frameBytes, err := io.ReadAll(part)
	if err != nil {
		return nil, err
	}
	return frameBytes, nil
}
