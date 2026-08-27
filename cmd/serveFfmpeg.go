package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/F2077/go-pubsub/pubsub"
	"github.com/ngyewch/go-mjpeg"
	"github.com/samber/oops"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func doServeFfmpeg(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.StringArg(inputFileArg.Name)
	if inputPath == "" {
		return fmt.Errorf("input-file is required")
	}
	streamLoop := cmd.Int(streamLoopFlag.Name)
	listenAddr := cmd.String(listenAddrFlag.Name)

	var ffmpegArgs []string
	ffmpegArgs = append(ffmpegArgs,
		"-hide_banner",
	)
	if streamLoop != 0 {
		ffmpegArgs = append(ffmpegArgs, "-stream_loop", strconv.Itoa(streamLoop))
	}
	ffmpegArgs = append(ffmpegArgs,
		"-i", inputPath,
		"-c:v", "mjpeg",
		"-an",
		"-f", "mjpeg",
		"pipe:1",
	)
	command := exec.Command("ffmpeg", ffmpegArgs...)
	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	defer func(stdoutPipe io.ReadCloser) {
		_ = stdoutPipe.Close()
	}(stdoutPipe)
	err = command.Start()
	if err != nil {
		return err
	}

	broker, err := pubsub.NewBroker[[]byte]()
	if err != nil {
		return err
	}
	publisher := pubsub.NewPublisher[[]byte](broker)

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		mjpegStreamDecoder := mjpeg.NewRawStreamDecoder(stdoutPipe)
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
