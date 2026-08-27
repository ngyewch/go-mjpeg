package mjpeg

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/jpeg"
	"io"
)

const (
	jpegMarkerByte = 0xff
	jpegSOI        = 0xd8
	jpegEOI        = 0xd9
	jpegSOS        = 0xda
)

// RawStreamDecoder is a raw MJPEG stream decoder.
type RawStreamDecoder struct {
	r io.Reader
}

// NewRawStreamDecoder creates a new RawStreamDecoder.
func NewRawStreamDecoder(r io.Reader) *RawStreamDecoder {
	return &RawStreamDecoder{
		r: r,
	}
}

// Next returns the next MJPEG frame as a byte array.
func (decoder *RawStreamDecoder) Next(ctx context.Context) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var markerBytes [2]byte
	frameBuffer := bytes.NewBuffer(nil)

	// expect SOI
	_, err := io.ReadFull(decoder.r, markerBytes[:])
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(markerBytes[:], []byte{jpegMarkerByte, jpegSOI}) {
		return nil, fmt.Errorf("SOI not found")
	}
	frameBuffer.Write(markerBytes[:])

	// loop through segments
	for {
		_, err = io.ReadFull(decoder.r, markerBytes[:])
		if err != nil {
			return nil, err
		}
		if markerBytes[0] != jpegMarkerByte {
			return nil, fmt.Errorf("marker not found")
		}
		frameBuffer.Write(markerBytes[:])
		markerType := markerBytes[1]
		if markerType == jpegEOI {
			frameBytes := frameBuffer.Bytes()
			frameReader := bytes.NewReader(frameBytes)
			_, err = jpeg.Decode(frameReader)
			if err != nil {
				return nil, err
			}
			return frameBytes, nil
		} else if markerType == jpegSOI {
			return nil, fmt.Errorf("unexpected SOI")
		} else if (markerType >= 0xd0) && (markerType <= 0xd7) { // restart
			continue
		}
		_, err = io.ReadFull(decoder.r, markerBytes[:])
		if err != nil {
			return nil, err
		}
		frameBuffer.Write(markerBytes[:])
		segmentLength := binary.BigEndian.Uint16(markerBytes[:])
		segmentBuffer := make([]byte, segmentLength-2)
		_, err = io.ReadFull(decoder.r, segmentBuffer)
		if err != nil {
			return nil, err
		}
		frameBuffer.Write(segmentBuffer)
		if markerType == jpegSOS {
			var byteBuffer [1]byte
			for {
				_, err := io.ReadFull(decoder.r, byteBuffer[:])
				if err != nil {
					return nil, err
				}
				_, err = frameBuffer.Write(byteBuffer[:])
				if err != nil {
					return nil, err
				}
				if byteBuffer[0] == jpegMarkerByte {
					_, err := io.ReadFull(decoder.r, byteBuffer[:])
					if err != nil {
						return nil, err
					}
					_, err = frameBuffer.Write(byteBuffer[:])
					if err != nil {
						return nil, err
					}
					if byteBuffer[0] == jpegEOI {
						frameBytes := frameBuffer.Bytes()
						frameReader := bytes.NewReader(frameBytes)
						_, err = jpeg.Decode(frameReader)
						if err != nil {
							return nil, err
						}
						return frameBytes, nil
					}
				}
			}
		}
	}
}
