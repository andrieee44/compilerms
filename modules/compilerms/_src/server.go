package main

import (
	"log/slog"
	"net/http"
)

func newServer(cfg *config) *http.Server {
	var (
		mux      *http.ServeMux
		langName string
		langCfg  *langConfig
		srv      http.Server
	)

	mux = http.NewServeMux()
	mux.Handle("GET /", newVersionsHandler(cfg))

	for langName, langCfg = range cfg.Languages {
		mux.Handle("GET /"+langName, newLiveHandler(cfg.WS, langCfg))
		mux.Handle("POST /"+langName, newBatchHandler(cfg.HTTP, langCfg))
	}

	srv = http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           newCORSMiddleware(cfg.HTTP)(mux),
		IdleTimeout:       cfg.HTTP.IdleTimeout.Duration,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
		Protocols:         new(http.Protocols),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout.Duration,
		ReadTimeout:       cfg.HTTP.ReadTimeout.Duration,
		WriteTimeout:      cfg.HTTP.WriteTimeout.Duration,
	}

	srv.Protocols.SetHTTP1(true)

	return &srv
}

func startServer(cfg *config) error {
	slog.Info("starting compiler microservice")

	return newServer(cfg).ListenAndServe()
}
