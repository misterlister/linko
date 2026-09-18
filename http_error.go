package main

import (
	"context"
	"net/http"
	"slices"
)

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logContextKey).(*LogContext); ok {
		logCtx.Error = err
	}

	errMsg := err.Error()

	sensitiveStatuses := []int{401, 403, 500}

	if slices.Contains(sensitiveStatuses, status) {
		errMsg = http.StatusText(status)
	}
	http.Error(w, errMsg, status)
}
