package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"

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

	mjpegStreamDecoder := mjpeg.NewRawStreamDecoder(stdoutPipe)

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
