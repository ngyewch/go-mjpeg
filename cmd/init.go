package main

import (
	"log/slog"
	"os"

	"github.com/jwalton/go-supportscolor"
	"github.com/phsym/console-slog"
)

func init() {
	initLogging()
}

func initLogging() {
	logLevel := slog.LevelInfo
	_ = logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	addSource := os.Getenv("LOG_ADD_SOURCE") == "true"
	noColor := os.Getenv("NO_COLOR") == "true"

	supportsColor := supportscolor.Stderr().SupportsColor
	consoleHandler := console.NewHandler(os.Stderr, &console.HandlerOptions{
		Level:     logLevel,
		AddSource: addSource,
		NoColor:   noColor || !supportsColor,
	})
	logger := slog.New(consoleHandler)
	slog.SetDefault(logger)
}
