package p2poolgo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testManifest(sha string) *Manifest {
	return &Manifest{
		Binaries: []Binary{
			{Platform: runtime.GOOS, Version: "v1", Url: "http://x/a.zip", Sha256: sha},
		},
		TrustedPeers:   []string{"t1"},
		BootstrapPeers: []string{"b1"},
	}
}

func okFetch(m *Manifest) func(string, interface{}) error {
	return func(url string, target interface{}) error {
		*(target.(*Manifest)) = *m
		return nil
	}
}

func failFetch(string, interface{}) error { return errors.New("offline") }

func TestLoadManifestFetchWritesCache(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "sub", "manifest.json")
	m, err := LoadManifest("http://x", cache, okFetch(testManifest("abc")), func(Binary) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Binaries) != 1 {
		t.Fatalf("unexpected manifest %+v", m)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("cache not written: %v", err)
	}
}

func TestLoadManifestOfflineUsesValidCache(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "manifest.json")
	archive := filepath.Join(dir, "a.zip")
	data := []byte("zipdata")
	os.WriteFile(archive, data, 0644)
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	af := func(Binary) string { return archive }
	if _, err := LoadManifest("http://x", cache, okFetch(testManifest(hexsum)), af); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest("http://x", cache, failFetch, af)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.TrustedPeers) != 1 || m.TrustedPeers[0] != "t1" {
		t.Fatalf("unexpected manifest %+v", m)
	}
}

func TestLoadManifestOfflineRejectsCacheWithoutArchive(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "manifest.json")
	af := func(Binary) string { return filepath.Join(dir, "missing.zip") }
	if _, err := LoadManifest("http://x", cache, okFetch(testManifest("abc")), af); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest("http://x", cache, failFetch, af); err == nil {
		t.Fatal("expected error")
	}
	// Archive present but wrong hash is also rejected.
	archive := filepath.Join(dir, "a.zip")
	os.WriteFile(archive, []byte("other"), 0644)
	if _, err := LoadManifest("http://x", cache, failFetch, func(Binary) string { return archive }); err == nil {
		t.Fatal("expected error for hash mismatch")
	}
}

func TestLoadManifestOfflineNoCacheErrors(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "manifest.json")
	_, err := LoadManifest("http://x", cache, failFetch, func(Binary) string { return "" })
	want := "Unable to download the P2Pool-Go manifest. Check your internet connection."
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}

func TestBinaryFor(t *testing.T) {
	m := &Manifest{Binaries: []Binary{
		{Platform: "windows", Version: "v1"},
		{Platform: "linux", Version: "v2"},
	}}
	if b, ok := m.BinaryFor("windows"); !ok || b.Version != "v1" {
		t.Fatalf("windows: %+v %v", b, ok)
	}
	if b, ok := m.BinaryFor("linux"); !ok || b.Version != "v2" {
		t.Fatalf("linux: %+v %v", b, ok)
	}
	if _, ok := m.BinaryFor("darwin"); ok {
		t.Fatal("darwin should not be found")
	}
}
