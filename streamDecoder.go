package mjpeg

import "context"

// StreamDecoder is the interface for decoding an MJPEG stream.
type StreamDecoder interface {
	// Next returns the next MJPEG frame as a byte array.
	Next(ctx context.Context) ([]byte, error)
}
