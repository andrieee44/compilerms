package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"

	"golang.org/x/sync/semaphore"
)

var sem *semaphore.Weighted = semaphore.NewWeighted(
	max(1, int64(runtime.NumCPU())/3),
)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx context.Context
			err error
		)

		ctx = r.Context()

		err = sem.Acquire(ctx, 1)
		if err != nil {
			slog.Error(r.URL.String(), "method", r.Method, "error", err)

			return
		}

		defer sem.Release(1)

		next.ServeHTTP(w, r)
	})
}
