// Package documents stores and retrieves documents in Cloud Storage.
//
// The module has a client and a service but, for now, no handler: the HTTP
// layer is added on top of Service in its own change. It has no repository
// either, because it owns no database tables. Cloud Storage is reached through
// client/, as the standard does for any external API.
package documents

import (
	"context"
	"fmt"

	"github.com/aaronrood-g2o/pr-examples/internal/modules/documents/client"
	"github.com/aaronrood-g2o/pr-examples/internal/modules/documents/service"
)

// Module exposes the documents module's capabilities to the application.
type Module struct {
	Service *service.Service
}

// NewModule wires the Cloud Storage client for bucket into the service.
func NewModule(ctx context.Context, bucket string) (*Module, error) {
	gcs, err := client.New(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("documents module: %w", err)
	}
	return &Module{Service: service.New(gcs)}, nil
}
