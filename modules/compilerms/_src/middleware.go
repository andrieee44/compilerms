package main

import (
	"net/http"

	"github.com/rs/cors"
)

func newCORSMiddleware(httpCfg *httpConfig) func(http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedHeaders: []string{"Content-Type"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedOrigins: httpCfg.AllowedOrigins,
	}).Handler
}
