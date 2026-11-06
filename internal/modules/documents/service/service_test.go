package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"
)

// stubStorage is an in-memory Storage that records calls.
type stubStorage struct {
	objects map[string]stubObject
	// err, when set, is returned by every call.
	err   error
	calls int
}

type stubObject struct {
	contentType string
	data        string
}

func newStub() *stubStorage {
	return &stubStorage{objects: map[string]stubObject{}}
}

func (s *stubStorage) Read(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	s.calls++
	if s.err != nil {
		return nil, "", 0, s.err
	}
	obj, ok := s.objects[key]
	if !ok {
		return nil, "", 0, fmt.Errorf("stub: %s: %w", key, fs.ErrNotExist)
	}
	return io.NopCloser(strings.NewReader(obj.data)), obj.contentType, int64(len(obj.data)), nil
}

func (s *stubStorage) Write(_ context.Context, key, contentType string, body io.Reader) error {
	s.calls++
	if s.err != nil {
		return s.err
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.objects[key] = stubObject{contentType: contentType, data: string(b)}
	return nil
}

func TestWriteThenRead(t *testing.T) {
	svc := New(newStub())
	if err := svc.Write(t.Context(), "letters/hello.txt", "text/plain; charset=utf-8", strings.NewReader("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	doc, err := svc.Read(t.Context(), "letters/hello.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer func() { _ = doc.Body.Close() }()
	b, _ := io.ReadAll(doc.Body)
	if string(b) != "hello" || doc.Size != 5 || doc.Key != "letters/hello.txt" ||
		doc.ContentType != "text/plain; charset=utf-8" {
		t.Errorf("got %+v with body %q", doc, b)
	}
}

func TestWriteDefaultsAnEmptyContentType(t *testing.T) {
	stub := newStub()
	if err := New(stub).Write(t.Context(), "a.bin", "", strings.NewReader("x")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := stub.objects["a.bin"].contentType; got != DefaultContentType {
		t.Errorf("content type %q, want %q", got, DefaultContentType)
	}
}

func TestWriteRejectsAMalformedContentType(t *testing.T) {
	stub := newStub()
	err := New(stub).Write(t.Context(), "a.txt", "text/", strings.NewReader("x"))
	if !errors.Is(err, ErrInvalidContentType) {
		t.Errorf("err %v, want ErrInvalidContentType", err)
	}
	if stub.calls != 0 {
		t.Errorf("storage called %d times for a rejected write", stub.calls)
	}
}

func TestReadMissingIsNotFound(t *testing.T) {
	if _, err := New(newStub()).Read(t.Context(), "gone.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err %v, want ErrNotFound", err)
	}
}

func TestStorageFailuresAreNotNotFound(t *testing.T) {
	stub := newStub()
	stub.err = errors.New("permission denied")
	svc := New(stub)

	if _, err := svc.Read(t.Context(), "a.txt"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("Read err %v, want a failure that is not ErrNotFound", err)
	}
	if err := svc.Write(t.Context(), "a.txt", "", strings.NewReader("x")); err == nil {
		t.Error("Write succeeded though storage failed")
	}
}

func TestInvalidKeysAreRejectedBeforeStorage(t *testing.T) {
	keys := map[string]string{
		"empty":            "",
		"too long":         strings.Repeat("a", maxKeyBytes+1),
		"invalid UTF-8":    "bad\xffkey",
		"control char":     "line\nbreak",
		"leading slash":    "/abs.txt",
		"trailing slash":   "dir/",
		"double slash":     "a//b",
		"dot segment":      "a/./b",
		"dot-dot segment":  "../etc/passwd",
		"lone dot-dot":     "..",
		"dot-dot mid-path": "a/../b",
	}
	for name, key := range keys {
		t.Run(name, func(t *testing.T) {
			stub := newStub()
			svc := New(stub)
			if _, err := svc.Read(t.Context(), key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Read err %v, want ErrInvalidKey", err)
			}
			if err := svc.Write(t.Context(), key, "", strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Write err %v, want ErrInvalidKey", err)
			}
			if stub.calls != 0 {
				t.Errorf("storage called %d times for an invalid key", stub.calls)
			}
		})
	}
}

func TestValidKeysAreAccepted(t *testing.T) {
	for _, key := range []string{
		"a",
		"letters/2026/hello.txt",
		"names with spaces/résumé.pdf",
		"dots..in.name",
		strings.Repeat("a", maxKeyBytes),
	} {
		if err := validateKey(key); err != nil {
			t.Errorf("validateKey(%q) = %v, want nil", key, err)
		}
	}
}
