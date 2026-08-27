package mjpeg

import (
	"context"
	"net/http"

	"github.com/F2077/go-pubsub/pubsub"
)

// Proxy is an http.Handler for serving an MJPEG stream.
type Proxy struct {
	streamDecoder StreamDecoder
	broker        *pubsub.Broker[[]byte]
	publisher     *pubsub.Publisher[[]byte]
	handler       *MultipartMixedReplaceHandler
}

// NewProxy creates a new Proxy.
func NewProxy(streamDecoder StreamDecoder) *Proxy {
	broker, err := pubsub.NewBroker[[]byte]()
	if err != nil {
		return nil
	}
	publisher := pubsub.NewPublisher[[]byte](broker)
	handler := NewMultipartMixedReplaceHandler("image/jpeg",
		func() (<-chan []byte, <-chan error, func() error, error) {
			subscriber := pubsub.NewSubscriber[[]byte](broker)
			subscription, err := subscriber.Subscribe("frames")
			if err != nil {
				_ = subscriber.Close()
				return nil, nil, nil, err
			}
			closer := func() error {
				_ = subscription.Close()
				_ = subscriber.Close()
				return nil
			}
			return subscription.Ch, subscription.ErrCh, closer, nil
		})
	return &Proxy{
		streamDecoder: streamDecoder,
		broker:        broker,
		publisher:     publisher,
		handler:       handler,
	}
}

// Run starts the proxy stream.
func (proxy *Proxy) Run(ctx context.Context) error {
	for {
		frameBytes, err := proxy.streamDecoder.Next(ctx)
		if err != nil {
			return err
		}
		err = proxy.publisher.Publish("frames", frameBytes)
		if err != nil {
			return err
		}
	}
}

// HTTPHandler returns the http.Handler
func (proxy *Proxy) HTTPHandler() http.Handler {
	return proxy.handler
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy.handler.ServeHTTP(w, r)
}
