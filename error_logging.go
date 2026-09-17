package main

import (
	"errors"
	"fmt"
	"log/slog"

	"boot.dev/linko/internal/linkoerr"
	pkgerr "github.com/pkg/errors"
)

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
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
