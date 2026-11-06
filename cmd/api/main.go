// Command api runs the pr-examples document API.
//
// For now it only loads configuration, which proves the environment is
// complete. Document storage and the HTTP server are wired in here as they
// are added.
package main

import (
	"log/slog"
	"os"

	"github.com/aaronrood-g2o/pr-examples/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.Info("configuration loaded", "documentsBucket", cfg.DocumentsBucket)
	return nil
}
