package p2poolgo

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func serveZip(t *testing.T, data []byte) Binary {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	sum := sha256.Sum256(data)
	return Binary{Platform: "x", Version: "v1", Url: srv.URL + "/node.zip", Sha256: hex.EncodeToString(sum[:])}
}

func TestInstallUnpacksToCurrent(t *testing.T) {
	data := makeZip(t, map[string]string{
		exeName():               "bin",
		"web/static/index.html": "<html></html>",
	})
	b := serveZip(t, data)
	binDir := filepath.Join(t.TempDir(), "p2pool-go-bin")
	exe, err := Install(b, binDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(binDir, "current", exeName()); exe != want {
		t.Fatalf("exe = %s, want %s", exe, want)
	}
	if _, err := os.Stat(filepath.Join(binDir, "current", "web", "static", "index.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(binDir, b.Sha256+".zip")); err != nil {
		t.Fatalf("archive not kept: %v", err)
	}
}

func TestInstallMissingExeErrors(t *testing.T) {
	data := makeZip(t, map[string]string{"web/static/index.html": "x"})
	b := serveZip(t, data)
	_, err := Install(b, filepath.Join(t.TempDir(), "bin"))
	if err == nil || !strings.Contains(err.Error(), "p2pool-go-vtc") {
		t.Fatalf("expected error mentioning p2pool-go-vtc, got %v", err)
	}
}
