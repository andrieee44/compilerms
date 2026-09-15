package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/andrieee44/compilerms/modules/compilerms/_src/compilerms/compilers"
)

func NewCompilerHandler[T any](
	compilerFn func(context.Context, T) (compilers.Output, error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			decoder *json.Decoder
			opts    T
			output  compilers.Output
			reason  string
			err     error
		)

		decoder = json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		err = decoder.Decode(&opts)
		if err != nil {
			slog.Error(r.URL.String(), "method", r.Method, "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		}

		output, err = compilerFn(r.Context(), opts)
		switch {
		case errors.Is(err, compilers.ErrCompiler):
			reason = compilers.ErrCompiler.Error()
		case errors.Is(err, compilers.ErrProgram):
			reason = compilers.ErrProgram.Error()
		case errors.Is(err, compilers.ErrBadRequest):
			slog.Error(r.URL.String(), "method", r.Method, "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		case err != nil:
			slog.Error(r.URL.String(), "method", r.Method, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(struct {
			Output string `json:"output"`
			Status int    `json:"status"`
			Reason string `json:"reason,omitempty"`
		}{
			Output: output.Output,
			Status: output.Status,
			Reason: reason,
		})
		if err != nil {
			slog.Error(r.URL.String(), "method", r.Method, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}
