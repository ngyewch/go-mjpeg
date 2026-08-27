package mjpeg

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// RetryingURLStreamDecoder is a retrying MJPEG URL stream decoder.
type RetryingURLStreamDecoder struct {
	url                 string
	httpClient          *http.Client
	httpRequestModifier func(httpRequest *http.Request)
	retryDelay          time.Duration
	disconnectedFrame   []byte
	urlStreamDecoder    *URLStreamDecoder
}

// NewRetryingURLStreamDecoder creates a new RetryingURLStreamDecoder.
func NewRetryingURLStreamDecoder(url string, httpClient *http.Client, httpRequestModifier func(httpRequest *http.Request), retryDelay time.Duration, disconnectedFrame []byte) *RetryingURLStreamDecoder {
	return &RetryingURLStreamDecoder{
		url:                 url,
		httpClient:          httpClient,
		httpRequestModifier: httpRequestModifier,
		retryDelay:          retryDelay,
		disconnectedFrame:   disconnectedFrame,
	}
}

// Close closes the decoder.
func (decoder *RetryingURLStreamDecoder) Close() error {
	if decoder.urlStreamDecoder != nil {
		_ = decoder.urlStreamDecoder.Close()
		decoder.urlStreamDecoder = nil
	}
	return nil
}

// Next returns the next MJPEG frame as a byte array.
func (decoder *RetryingURLStreamDecoder) Next(ctx context.Context) ([]byte, error) {
	for {
		for decoder.urlStreamDecoder == nil {
			urlStreamDecoder, err := NewURLStreamDecoder(decoder.url, decoder.httpClient, decoder.httpRequestModifier)
			if err != nil {
				slog.Error("error connecting to stream",
					slog.String("url", decoder.url),
					slog.Any("err", err),
				)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(decoder.retryDelay):
					if decoder.disconnectedFrame != nil {
						return decoder.disconnectedFrame, nil
					}
				}
			} else {
				slog.Info("connected to stream",
					slog.String("url", decoder.url),
				)
				decoder.urlStreamDecoder = urlStreamDecoder
				break
			}
		}
		frameBytes, err := decoder.urlStreamDecoder.Next(ctx)
		if err != nil {
			slog.Error("error reading frame",
				slog.String("url", decoder.url),
				slog.Any("err", err),
			)
			_ = decoder.urlStreamDecoder.Close()
			decoder.urlStreamDecoder = nil
			if decoder.disconnectedFrame != nil {
				return decoder.disconnectedFrame, nil
			}
		} else {
			return frameBytes, nil
		}
	}
}
