package main

import (
	"context"
	"fmt"
	"net/http"

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

	proxy := mjpeg.NewProxy(mjpegStreamDecoder)

	errGroup, errGroupCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		return proxy.Run(errGroupCtx)
	})

	errGroup.Go(func() error {
		httpServer := &http.Server{
			Addr:    listenAddr,
			Handler: proxy,
		}
		err := httpServer.ListenAndServe()
		if err != nil {
			return oops.Wrapf(err, "error starting http server")
		}
		return nil
	})

	err := errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}
