package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

type closeFunc func() error

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			logger.Info("Served request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("client_ip", r.RemoteAddr),
			)
		})
	}
}

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	noOpCloseFunc := func() error { return nil }

	if logFile != "" {
		debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
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
			Level: slog.LevelInfo,
		})

		logger := slog.New(slog.NewMultiHandler(
			debugHandler,
			infoHandler,
		))

		return logger, closeFunc, nil
	}

	return slog.New(slog.NewTextHandler(os.Stderr, nil)), noOpCloseFunc, nil
}
