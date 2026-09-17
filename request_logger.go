package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const logContextKey contextKey = "log_context"

type LogContext struct {
	Username string
	Error    error
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spyWriter := &spyResponseWriter{ResponseWriter: w}
			spyReader := &spyReadCloser{ReadCloser: r.Body}
			r.Body = spyReader
			start := time.Now()
			logCtx := &LogContext{Username: ""}
			r = r.WithContext(context.WithValue(r.Context(), logContextKey, logCtx))
			next.ServeHTTP(spyWriter, r)

			logAttrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("client_ip", r.RemoteAddr),
				slog.Duration("duration", time.Since(start)),
				slog.Int("request_body_bytes", spyReader.bytesRead),
				slog.Int("response_status", spyWriter.statusCode),
				slog.Int("response_body_bytes", spyWriter.bytesWritten),
			}

			if logCtx.Username != "" {
				logAttrs = append(logAttrs, slog.String("user", logCtx.Username))
			}

			if logCtx.Error != nil {
				logAttrs = append(logAttrs, slog.Any("error", logCtx.Error))
			}

			logger.Info("Served request", logAttrs...)
		})
	}
}
