package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

type closeFunc func() error

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spyWriter := &spyResponseWriter{ResponseWriter: w}
			spyReader := &spyReadCloser{ReadCloser: r.Body}
			r.Body = spyReader
			start := time.Now()
			next.ServeHTTP(spyWriter, r)

			logger.Info("Served request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("client_ip", r.RemoteAddr),
				slog.Duration("duration", time.Since(start)),
				slog.Int("request_body_bytes", spyReader.bytesRead),
				slog.Int("response_status", spyWriter.statusCode),
				slog.Int("response_body_bytes", spyWriter.bytesWritten),
			)
		})
	}
}

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

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}

		if multiErr, ok := errors.AsType[multiError](err); ok {
			errs := multiErr.Unwrap()
			var errAttrSlice []slog.Attr

			for i := range errs {
				errName := fmt.Sprintf("error_%d", (i + 1))
				newAttr := slog.GroupAttrs(errName, errorAttrs(errs[i])...)
				errAttrSlice = append(errAttrSlice, newAttr)
			}

			return slog.GroupAttrs("errors", errAttrSlice...)
		}

		return slog.GroupAttrs("error", errorAttrs(err)...)
	}
	return a
}

func errorAttrs(err error) []slog.Attr {
	attributes := []slog.Attr{
		{
			Key:   "message",
			Value: slog.StringValue(err.Error()),
		},
	}

	attributes = append(attributes, linkoerr.Attrs(err)...)

	if stackErr, ok := errors.AsType[stackTracer](err); ok {
		attributes = append(attributes, slog.Attr{
			Key:   "stack_trace",
			Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
		},
		)
	}
	return attributes
}
