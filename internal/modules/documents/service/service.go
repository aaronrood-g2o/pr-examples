// Package service holds the documents module's business rules: which keys and
// content types are acceptable, and what a missing document means.
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Errors callers can match with errors.Is. The HTTP handler maps these to
// status codes; anything else is an internal failure.
var (
	ErrNotFound           = errors.New("document not found")
	ErrInvalidKey         = errors.New("invalid document key")
	ErrInvalidContentType = errors.New("invalid content type")
)

// DefaultContentType is stored when a write names no content type.
const DefaultContentType = "application/octet-stream"

// maxKeyBytes is Cloud Storage's limit on an object name.
const maxKeyBytes = 1024

// Storage is what the service needs from the bucket. client.GCS satisfies it.
// A missing object on Read is an error wrapping fs.ErrNotExist.
type Storage interface {
	Read(ctx context.Context, key string) (body io.ReadCloser, contentType string, size int64, err error)
	Write(ctx context.Context, key, contentType string, body io.Reader) error
}

// Document is one stored document, open for reading. The caller closes Body.
type Document struct {
	Key         string
	ContentType string
	// Size is the length in bytes, or -1 when storage did not report one.
	Size int64
	Body io.ReadCloser
}

// Service reads and writes documents.
type Service struct {
	storage Storage
}

// New returns a Service backed by storage.
func New(storage Storage) *Service {
	return &Service{storage: storage}
}

// Read opens the document at key. A key that does not exist returns
// ErrNotFound. A malformed key returns ErrInvalidKey without touching storage.
func (s *Service) Read(ctx context.Context, key string) (*Document, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	body, contentType, size, err := s.storage.Read(ctx, key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, key)
		}
		return nil, fmt.Errorf("reading document %q: %w", key, err)
	}
	return &Document{Key: key, ContentType: contentType, Size: size, Body: body}, nil
}

// Write stores body at key, replacing any existing document there. An empty
// contentType is stored as DefaultContentType. A malformed key or content
// type is rejected before anything is uploaded.
func (s *Service) Write(ctx context.Context, key, contentType string, body io.Reader) error {
	if err := validateKey(key); err != nil {
		return err
	}
	contentType, err := normalizeContentType(contentType)
	if err != nil {
		return err
	}
	if err := s.storage.Write(ctx, key, contentType, body); err != nil {
		return fmt.Errorf("writing document %q: %w", key, err)
	}
	return nil
}

// validateKey accepts keys that are safe to use as object names and that the
// HTTP layer can route to unambiguously: non-empty, valid UTF-8, within Cloud
// Storage's length limit, no control characters, no leading slash, and no
// empty, "." or ".." path segments.
func validateKey(key string) error {
	switch {
	case key == "":
		return fmt.Errorf("%w: key is empty", ErrInvalidKey)
	case len(key) > maxKeyBytes:
		return fmt.Errorf("%w: key is longer than %d bytes", ErrInvalidKey, maxKeyBytes)
	case !utf8.ValidString(key):
		return fmt.Errorf("%w: key is not valid UTF-8", ErrInvalidKey)
	case strings.IndexFunc(key, unicode.IsControl) >= 0:
		return fmt.Errorf("%w: key contains a control character", ErrInvalidKey)
	}
	for _, segment := range strings.Split(key, "/") {
		switch segment {
		case "":
			return fmt.Errorf("%w: key has an empty path segment", ErrInvalidKey)
		case ".", "..":
			return fmt.Errorf("%w: key has a %q path segment", ErrInvalidKey, segment)
		}
	}
	return nil
}

// normalizeContentType defaults an empty content type and rejects one that
// does not parse as a media type.
func normalizeContentType(contentType string) (string, error) {
	if contentType == "" {
		return DefaultContentType, nil
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %v", ErrInvalidContentType, contentType, err)
	}
	return mime.FormatMediaType(mediaType, params), nil
}
