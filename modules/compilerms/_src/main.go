package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"golang.org/x/sync/semaphore"
)

var sem *semaphore.Weighted

func run() error {
	var (
		cfgPath string
		cfg     *config
		err     error
	)

	slog.SetDefault(slog.New(tint.NewTextHandler(os.Stderr, nil)))

	if len(os.Args) < 2 {
		return errors.New("missing configuration file path argument")
	}

	cfgPath = os.Args[1]

	cfg, err = newConfig(cfgPath)
	if err != nil {
		return err
	}

	sem = semaphore.NewWeighted(cfg.MaxRunningSteps)

	err = testServer(cfg)
	if err != nil {
		return err
	}

	return startServer(cfg)
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `compilerms: %v

Usage:   compilerms <CONFIG_FILE>
Example: compilerms ~/.config/compilerms/config.jsonnet
`, err)

		os.Exit(1)
	}
}
