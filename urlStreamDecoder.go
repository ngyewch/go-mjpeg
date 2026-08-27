package mjpeg

import (
	"context"
	"net/http"
)

// URLStreamDecoder is a MJPEG URL stream decoder.
type URLStreamDecoder struct {
	httpResponse      *http.Response
	httpStreamDecoder *HTTPStreamDecoder
}

// NewURLStreamDecoder creates a new URLStreamDecoder.
func NewURLStreamDecoder(url string, httpClient *http.Client, httpRequestModifier func(httpRequest *http.Request)) (*URLStreamDecoder, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if httpRequestModifier != nil {
		httpRequestModifier(httpRequest)
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	httpStreamDecoder, err := NewHTTPStreamDecoderFromHTTPResponse(httpResponse)
	if err != nil {
		_ = httpResponse.Body.Close()
		return nil, err
	}
	return &URLStreamDecoder{
		httpResponse:      httpResponse,
		httpStreamDecoder: httpStreamDecoder,
	}, nil
}

// Close closes the decoder.
func (decoder *URLStreamDecoder) Close() error {
	_ = decoder.httpResponse.Body.Close()
	return nil
}

// Next returns the next MJPEG frame as a byte array.
func (decoder *URLStreamDecoder) Next(ctx context.Context) ([]byte, error) {
	return decoder.httpStreamDecoder.Next(ctx)
}
