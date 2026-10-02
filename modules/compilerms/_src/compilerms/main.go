package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/andrieee44/compilerms/modules/compilerms/_src/compilerms/compilers"
	"github.com/andrieee44/compilerms/modules/compilerms/_src/compilerms/handlers"
)

func run() error {
	var (
		address string
		mux     *http.ServeMux
		srv     *http.Server
		err     error
	)

	if len(os.Args) < 2 {
		return errors.New("missing argument <ADDRESS>")
	}

	address = os.Args[1]

	mux = http.NewServeMux()
	mux.Handle("POST /gcc", handlers.NewCompilerHandler(compilers.GCC))
	mux.Handle("POST /java", handlers.NewCompilerHandler(compilers.Java))
	mux.Handle("POST /sqlite3tmp", handlers.NewCompilerHandler(compilers.SQLite3tmp))

	srv = &http.Server{
		Addr:              address,
		Handler:           handlers.CORS(handlers.RateLimiter(mux)),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}

	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)

	slog.Info("Compiler Microservice Starting", "address", address)

	err = srv.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `compilerms: %v

Usage:   compilerms <ADDRESS>
Example: compilerms localhost:8080
`, err)

		os.Exit(1)
	}
}
