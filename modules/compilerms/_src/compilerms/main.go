package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/lmittmann/tint"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func run() error {
	var (
		cfg  *Config
		mux  *http.ServeMux
		name string
		lang *Language
		srv  *http.Server
		err  error
	)

	slog.SetDefault(slog.New(tint.NewTextHandler(os.Stderr, nil)))

	cfg, err = LoadConfig()
	if err != nil {
		return err
	}

	mux = http.NewServeMux()

	for name = range cfg.Languages {
		if !cfg.Languages[name].Enable {
			continue
		}

		lang = NewLanguage(name, cfg)

		err = lang.RunTests()
		if err != nil {
			return err
		}

		mux.HandleFunc(fmt.Sprintf("POST /%s", name), lang.Handler)
		mux.HandleFunc(fmt.Sprintf("GET /%s", name), lang.HandlerWS)
	}

	srv = &http.Server{
		Addr:              cfg.Global.HTTPServer.Address,
		Handler:           CORS(mux),
		IdleTimeout:       cfg.Global.HTTPServer.IdleTimeout,
		MaxHeaderBytes:    cfg.Global.HTTPServer.MaxHeaderBytes,
		ReadHeaderTimeout: cfg.Global.HTTPServer.ReadHeaderTimeout,
		ReadTimeout:       cfg.Global.HTTPServer.ReadTimeout,
		WriteTimeout:      cfg.Global.HTTPServer.WriteTimeout,
	}

	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)

	slog.Info("Compiler Microservice Starting", "address", srv.Addr)

	return srv.ListenAndServe()
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `compilerms: %v

Usage: COMPILERMS_CONFIG_PATH="/path/to/config.yaml" compilerms
`, err)

		os.Exit(1)
	}
}
