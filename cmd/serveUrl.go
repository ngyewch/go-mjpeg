package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/F2077/go-pubsub/pubsub"
	"github.com/ngyewch/go-mjpeg"
	"github.com/samber/oops"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func doServeURL(ctx context.Context, cmd *cli.Command) error {
	inputURL := cmd.StringArg(inputURLArg.Name)
	if inputURL == "" {
		return fmt.Errorf("input-url is required")
	}
	listenAddr := cmd.String(listenAddrFlag.Name)
	reconnectionDelay := cmd.Duration(reconnectionDelayFlag.Name)
	username := cmd.String(usernameFlag.Name)
	password := cmd.String(passwordFlag.Name)

	broker, err := pubsub.NewBroker[[]byte]()
	if err != nil {
		return err
	}
	publisher := pubsub.NewPublisher[[]byte](broker)

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		mjpegStreamDecoder := mjpeg.NewRetryingURLStreamDecoder(
			inputURL,
			nil,
			func(httpRequest *http.Request) {
				if (username != "") && (password != "") {
					httpRequest.SetBasicAuth(username, password)
				}
			},
			reconnectionDelay,
			nil)
		defer func(mjpegStreamDecoder *mjpeg.RetryingURLStreamDecoder) {
			_ = mjpegStreamDecoder.Close()
		}(mjpegStreamDecoder)
		for {
			frameBytes, err := mjpegStreamDecoder.Next(errGroupCtx)
			if err != nil {
				return err
			}
			err = publisher.Publish("frames", frameBytes)
			if err != nil {
				return err
			}
		}
	})

	errGroup.Go(func() error {
		handler := mjpeg.NewMultipartMixedReplaceHandler("image/jpeg",
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
		httpServer := &http.Server{
			Addr:    listenAddr,
			Handler: handler,
		}
		err = httpServer.ListenAndServe()
		if err != nil {
			return oops.Wrapf(err, "error starting http server")
		}
		return nil
	})

	err = errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}
