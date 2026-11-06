package client

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
)

// recorded is what the fake server saw of the last request.
type recorded struct {
	method string
	path   string
	query  map[string][]string
	// parts holds each multipart/related part's body, for uploads.
	parts []string
}

// fakeGCS stands in for the Cloud Storage JSON API. It answers every request
// with status and body, and records the request so a test can check what the
// client asked for.
func fakeGCS(t *testing.T, status int, contentType, body string) (*GCS, *recorded) {
	t.Helper()
	got := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.EscapedPath()
		got.query = r.URL.Query()
		got.parts = readParts(t, r)
		if status != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = fmt.Fprintf(w, `{"error":{"code":%d,"message":"x"}}`, status)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	g, err := New(t.Context(), "documents-bucket",
		option.WithEndpoint(srv.URL+"/storage/v1/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g, got
}

// readParts returns the bodies of a multipart/related request's parts, or nil
// for any other request.
func readParts(t *testing.T, r *http.Request) []string {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil
	}
	var parts []string
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return parts
		}
		if err != nil {
			t.Errorf("reading multipart body: %v", err)
			return parts
		}
		b, _ := io.ReadAll(p)
		parts = append(parts, string(b))
	}
}

func TestReadDownloadsTheObject(t *testing.T) {
	g, got := fakeGCS(t, http.StatusOK, "text/plain", "hello")
	body, contentType, size, err := g.Read(t.Context(), "letters/2026/hello.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer func() { _ = body.Close() }()
	b, _ := io.ReadAll(body)
	if string(b) != "hello" || size != int64(len(b)) || contentType != "text/plain" {
		t.Errorf("got body %q, size %d, content type %q", b, size, contentType)
	}
	if want := "/storage/v1/b/documents-bucket/o/letters%2F2026%2Fhello.txt"; got.path != want {
		t.Errorf("path %s, want %s", got.path, want)
	}
	if got.query["alt"][0] != "media" {
		t.Errorf("query %v, want alt=media", got.query)
	}
}

func TestReadMissingObjectIsNotExist(t *testing.T) {
	g, _ := fakeGCS(t, http.StatusNotFound, "", "")
	if _, _, _, err := g.Read(t.Context(), "gone.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err %v, want fs.ErrNotExist", err)
	}
}

func TestReadForbiddenIsNotNotExist(t *testing.T) {
	g, _ := fakeGCS(t, http.StatusForbidden, "", "")
	_, _, _, err := g.Read(t.Context(), "secret.txt")
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err %v, want a failure that is not fs.ErrNotExist", err)
	}
}

func TestWriteUploadsNameContentTypeAndBytes(t *testing.T) {
	g, got := fakeGCS(t, http.StatusOK, "application/json", `{"name":"letters/hello.txt"}`)
	if err := g.Write(t.Context(), "letters/hello.txt", "text/plain", strings.NewReader("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got.method != http.MethodPost {
		t.Errorf("method %s, want POST", got.method)
	}
	if want := "/upload/storage/v1/b/documents-bucket/o"; got.path != want {
		t.Errorf("path %s, want %s", got.path, want)
	}
	if len(got.parts) != 2 {
		t.Fatalf("got %d multipart parts, want metadata and media", len(got.parts))
	}
	if meta := got.parts[0]; !strings.Contains(meta, `"name":"letters/hello.txt"`) ||
		!strings.Contains(meta, `"contentType":"text/plain"`) {
		t.Errorf("metadata %s, want the name and content type", meta)
	}
	if got.parts[1] != "hello" {
		t.Errorf("media %q, want %q", got.parts[1], "hello")
	}
}

func TestWriteFailureIsReturned(t *testing.T) {
	g, _ := fakeGCS(t, http.StatusForbidden, "", "")
	if err := g.Write(t.Context(), "x.txt", "text/plain", strings.NewReader("x")); err == nil {
		t.Error("Write succeeded against a 403")
	}
}

func TestNewRefusesAnEmptyBucket(t *testing.T) {
	if _, err := New(t.Context(), "", option.WithoutAuthentication()); err == nil {
		t.Error("New accepted an empty bucket")
	}
}
