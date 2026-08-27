package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

var (
	version string

	listenAddrFlag = &cli.StringFlag{
		Name:    "listen-addr",
		Usage:   "listen address",
		Value:   ":8080",
		Sources: cli.EnvVars("LISTEN_ADDR"),
	}
	streamLoopFlag = &cli.IntFlag{
		Name:    "stream-loop",
		Usage:   "stream loop count",
		Value:   -1,
		Sources: cli.EnvVars("STREAM_LOOP"),
	}
	usernameFlag = &cli.StringFlag{
		Name:    "username",
		Usage:   "username",
		Sources: cli.EnvVars("USERNAME"),
	}
	passwordFlag = &cli.StringFlag{
		Name:    "password",
		Usage:   "password",
		Sources: cli.EnvVars("PASSWORD"),
	}
	reconnectionDelayFlag = &cli.DurationFlag{
		Name:    "reconnection-delay",
		Usage:   "reconnection delay",
		Value:   1 * time.Second,
		Sources: cli.EnvVars("RECONNECTION_DELAY"),
	}

	inputFileArg = &cli.StringArg{
		Name:      "input-file",
		UsageText: "(input-file)",
	}
	inputUrlArg = &cli.StringArg{
		Name:      "input-url",
		UsageText: "(input-url)",
	}

	app = &cli.Command{
		Name:    "go-mjpeg",
		Usage:   "go-mjpeg",
		Version: version,
		Commands: []*cli.Command{
			{
				Name: "serve",
				Commands: []*cli.Command{
					{
						Name:   "ffmpeg",
						Usage:  "ffmpeg",
						Action: doServeFfmpeg,
						Arguments: []cli.Argument{
							inputFileArg,
						},
						Flags: []cli.Flag{
							listenAddrFlag,
							streamLoopFlag,
						},
					},
					{
						Name:   "url",
						Usage:  "url",
						Action: doServeUrl,
						Arguments: []cli.Argument{
							inputUrlArg,
						},
						Flags: []cli.Flag{
							listenAddrFlag,
							usernameFlag,
							passwordFlag,
							reconnectionDelayFlag,
						},
					},
				},
			},
		},
	}
)

func main() {
	err := app.Run(context.Background(), os.Args)
	if err != nil {
		slog.Error("error",
			slog.Any("err", err),
		)
		os.Exit(1)
	}
}
