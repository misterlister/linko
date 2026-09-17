package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	noOpCloseFunc := func() error { return nil }

	if logFile != "" {
		debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
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

	return slog.New(slog.NewTextHandler(os.Stderr, nil)), noOpCloseFunc, nil
}
