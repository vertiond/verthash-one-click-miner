package miners

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func buildTestZip(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

func serveZip(t *testing.T, data []byte) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestEnsureArchiveDownloadsAndVerifies(t *testing.T) {
	data, sum := buildTestZip(t)
	srv, hits := serveZip(t, data)
	path := filepath.Join(t.TempDir(), "archive")
	if err := EnsureArchive(srv.URL+"/x.zip", sum, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Fatalf("expected 1 hit, got %d", n)
	}
}

func TestEnsureArchiveSkipsWhenPresent(t *testing.T) {
	data, sum := buildTestZip(t)
	srv, hits := serveZip(t, data)
	path := filepath.Join(t.TempDir(), "archive")
	for i := 0; i < 2; i++ {
		if err := EnsureArchive(srv.URL+"/x.zip", sum, path); err != nil {
			t.Fatal(err)
		}
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Fatalf("expected 1 hit, got %d", n)
	}
}

func TestEnsureArchiveRedownloadsOnMismatch(t *testing.T) {
	data, sum := buildTestZip(t)
	srv, hits := serveZip(t, data)
	path := filepath.Join(t.TempDir(), "archive")
	if err := os.WriteFile(path, []byte("junk"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureArchive(srv.URL+"/x.zip", sum, path); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Fatalf("expected 1 hit, got %d", n)
	}
}

func TestEnsureArchiveFailsOnBadFreshDownload(t *testing.T) {
	data, _ := buildTestZip(t)
	srv, _ := serveZip(t, data)
	path := filepath.Join(t.TempDir(), "archive")
	if err := EnsureArchive(srv.URL+"/x.zip", "00ff", path); err == nil {
		t.Fatal("expected error")
	}
}

func TestUnpackArchiveZip(t *testing.T) {
	data, _ := buildTestZip(t)
	dir := t.TempDir()
	archive := filepath.Join(dir, "archive")
	if err := os.WriteFile(archive, data, 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dest, "stale.txt")
	if err := os.WriteFile(stale, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := UnpackArchive("http://h/x.zip", archive, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "a", "b.txt")); err != nil {
		t.Fatalf("a/b.txt missing: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale file still present")
	}
}
