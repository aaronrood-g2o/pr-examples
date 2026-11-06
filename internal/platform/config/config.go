// Package config loads application configuration from the environment.
package config

import (
	"errors"
	"os"
)

// Config is the application's runtime configuration.
type Config struct {
	// DocumentsBucket is the Cloud Storage bucket documents are read from and
	// written to. Terraform creates it (infra/terraform/storage.tf) and reports
	// its name as the documents_bucket output.
	DocumentsBucket string
}

// Load reads Config from the environment. Every field is required: the
// application has no purpose without a bucket, so a missing one fails at
// startup rather than on the first request.
func Load() (Config, error) {
	cfg := Config{
		DocumentsBucket: os.Getenv("GCS_DOCUMENTS_BUCKET"),
	}
	if cfg.DocumentsBucket == "" {
		return Config{}, errors.New("config: GCS_DOCUMENTS_BUCKET is required")
	}
	return cfg, nil
}
