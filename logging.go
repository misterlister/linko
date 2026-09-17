package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	noOpCloseFunc := func() error { return nil }

	disableColour := !(isatty.IsCygwinTerminal(os.Stderr.Fd()) || isatty.IsTerminal(os.Stderr.Fd()))

	if logFile != "" {
		debugHandler := tint.NewTextHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
			NoColor:     disableColour,
		})

		file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, noOpCloseFunc, fmt.Errorf("failed to open log file: %w", err)
		}

		bufferedFile := bufio.NewWriterSize(file, 8192)

		closeFunc := func() error {
			err = bufferedFile.Flush()
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}

			return nil
		}

		infoHandler := slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		})

		logger := slog.New(slog.NewMultiHandler(
			debugHandler,
			infoHandler,
		))

		return logger, closeFunc, nil
	}

	return slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{
		NoColor: disableColour,
	})), noOpCloseFunc, nil
}
