package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/andrieee44/compilerms/modules/compilerms/_src/compilerms/compilers"
)

func GCC(w http.ResponseWriter, r *http.Request) {
	var (
		decoder *json.Decoder
		opts    compilers.GCCOpts
		output  compilers.Output
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

	output, err = compilers.GCC(r.Context(), opts)
	if err != nil {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)

		switch {
		case errors.Is(err, compilers.ErrBadRequest):
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		case errors.Is(err, compilers.ErrInternal):
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(output)
	if err != nil {
		slog.Error(r.URL.String(), "method", r.Method, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
