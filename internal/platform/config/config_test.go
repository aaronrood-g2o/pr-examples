package config

import "testing"

func TestLoadReadsTheBucket(t *testing.T) {
	t.Setenv("GCS_DOCUMENTS_BUCKET", "my-project-documents")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DocumentsBucket != "my-project-documents" {
		t.Errorf("DocumentsBucket %q", cfg.DocumentsBucket)
	}
}

func TestLoadRequiresTheBucket(t *testing.T) {
	t.Setenv("GCS_DOCUMENTS_BUCKET", "")
	if _, err := Load(); err == nil {
		t.Error("Load succeeded without GCS_DOCUMENTS_BUCKET")
	}
}
