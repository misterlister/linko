package main

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"gopkg.in/natefinch/lumberjack.v2"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	noOpCloseFunc := func() error { return nil }

	disableColour := !(isatty.IsCygwinTerminal(os.Stderr.Fd()) || isatty.IsTerminal(os.Stderr.Fd()))

	if logFile != "" {
		handlers := []slog.Handler{}

		handlers = append(handlers, tint.NewTextHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
			NoColor:     disableColour,
		}))

		logger := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    1,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(logger, &slog.HandlerOptions{
			ReplaceAttr: replaceAttr,
		}))

		closeFunc := func() error {
			return logger.Close()
		}

		multiLogger := slog.New(slog.NewMultiHandler(
			handlers...,
		))

		return multiLogger, closeFunc, nil
	}

	return slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{
		NoColor: disableColour,
	})), noOpCloseFunc, nil
}
