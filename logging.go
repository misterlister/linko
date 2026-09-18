package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"gopkg.in/natefinch/lumberjack.v2"
)

type closeFunc func() error

const Redacted string = "[REDACTED]"

var sensitiveKeys = []string{
	"password",
	"key",
	"apikey",
	"secret",
	"pin",
	"creditcardno",
	"user",
}

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

	if slices.Contains(sensitiveKeys, a.Key) {
		a.Value = slog.StringValue(Redacted)
		return a
	}

	if a.Value.Kind() == slog.KindString {
		parsedURL, err := url.Parse(a.Value.String())

		if err != nil {
			return a
		}

		userInfo := parsedURL.User

		if userInfo == nil {
			return a
		}

		username := userInfo.Username()

		_, passwordPresent := userInfo.Password()

		if !passwordPresent {
			return a
		}

		redactedUser := url.UserPassword(username, Redacted)
		parsedURL.User = redactedUser
		redactedURL := slog.StringValue(parsedURL.String())

		a.Value = redactedURL
	}

	return a
}
