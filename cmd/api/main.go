// Command api runs the pr-examples document API.
//
// For now it loads configuration and builds the Cloud Storage client, which
// proves the environment is complete and Application Default Credentials can
// be found. The document service and the HTTP server are wired in here as
// they are added.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aaronrood-g2o/pr-examples/internal/modules/documents/client"
	"github.com/aaronrood-g2o/pr-examples/internal/platform/config"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("startup failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, err := client.New(ctx, cfg.DocumentsBucket); err != nil {
		return err
	}
	slog.Info("documents storage client ready", "documentsBucket", cfg.DocumentsBucket)
	return nil
}
