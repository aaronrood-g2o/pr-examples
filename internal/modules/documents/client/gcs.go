// Package client talks to Cloud Storage on behalf of the documents module.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	storage "google.golang.org/api/storage/v1"
)

// GCS reads and writes objects in one Cloud Storage bucket through the JSON
// API, using Application Default Credentials unless opts say otherwise. On
// Cloud Run that is the runtime service account, which needs
// roles/storage.objectAdmin on the bucket (infra/terraform/storage.tf).
//
// It knows nothing about documents: it moves bytes under a key. Deciding which
// keys are acceptable is the service's job.
type GCS struct {
	objects *storage.ObjectsService
	bucket  string
}

// New returns a client for bucket. opts exist for tests, which point it at a
// local server.
func New(ctx context.Context, bucket string, opts ...option.ClientOption) (*GCS, error) {
	if bucket == "" {
		return nil, errors.New("gcs client: bucket is empty")
	}
	opts = append([]option.ClientOption{option.WithScopes(storage.DevstorageReadWriteScope)}, opts...)
	svc, err := storage.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcs client: creating storage client: %w", err)
	}
	return &GCS{objects: storage.NewObjectsService(svc), bucket: bucket}, nil
}

// Read starts a download of the live version of the object at key. The caller
// closes body. size is the object's length, or -1 when the response did not
// carry one. A missing object is an error wrapping fs.ErrNotExist.
func (g *GCS) Read(ctx context.Context, key string) (body io.ReadCloser, contentType string, size int64, err error) {
	resp, err := g.objects.Get(g.bucket, key).Context(ctx).Download()
	if err != nil {
		if isNotFound(err) {
			return nil, "", 0, fmt.Errorf("gcs client: gs://%s/%s: %w", g.bucket, key, fs.ErrNotExist)
		}
		return nil, "", 0, fmt.Errorf("gcs client: downloading gs://%s/%s: %w", g.bucket, key, err)
	}
	return resp.Body, resp.Header.Get("Content-Type"), resp.ContentLength, nil
}

// Write uploads body to key with the given content type, replacing any live
// version. The bucket is versioned, so a replaced version is kept as a
// noncurrent generation rather than lost.
func (g *GCS) Write(ctx context.Context, key, contentType string, body io.Reader) error {
	obj := &storage.Object{Name: key, ContentType: contentType}
	_, err := g.objects.Insert(g.bucket, obj).
		Media(body, googleapi.ContentType(contentType)).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("gcs client: uploading gs://%s/%s: %w", g.bucket, key, err)
	}
	return nil
}

func isNotFound(err error) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound
}
