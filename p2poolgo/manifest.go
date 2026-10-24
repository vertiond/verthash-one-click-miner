package p2poolgo

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

const ManifestURL = "https://raw.githubusercontent.com/vertiond/verthash-one-click-miner/master/p2pool-go.json"

type Binary struct {
	Platform string `json:"platform"` // runtime.GOOS value
	Version  string `json:"version"`
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
}

type Manifest struct {
	Binaries       []Binary `json:"binaries"`
	TrustedPeers   []string `json:"trustedPeers"`
	BootstrapPeers []string `json:"bootstrapPeers"`
}

// BinaryFor returns the binary entry for the given runtime.GOOS value.
func (m *Manifest) BinaryFor(goos string) (Binary, bool) {
	for _, b := range m.Binaries {
		if b.Platform == goos {
			return b, true
		}
	}
	return Binary{}, false
}

// validSha256 reports whether s is exactly 64 hex characters. Archive paths
// are built from it, so nothing else may get that far.
func validSha256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// valid reports whether every binary entry carries a well-formed sha256.
func (m *Manifest) valid() bool {
	for _, b := range m.Binaries {
		if !validSha256(b.Sha256) {
			return false
		}
	}
	return true
}

var errManifestUnavailable = errors.New("Unable to download the P2Pool-Go manifest. Check your internet connection.")

// LoadManifest fetches the manifest and caches it at cachePath. If the fetch
// fails, the cached copy is used, but only when the archive it refers to for
// this platform is already on disk (archiveFor) and matches its sha256.
func LoadManifest(url, cachePath string, fetch func(string, interface{}) error, archiveFor func(Binary) string) (*Manifest, error) {
	m := &Manifest{}
	fetchErr := fetch(url, m)
	if fetchErr == nil && !m.valid() {
		fetchErr = errors.New("the manifest has an invalid sha256")
	}
	if fetchErr == nil {
		if err := writeCache(cachePath, m); err != nil {
			logging.Warnf("Unable to cache the P2Pool-Go manifest: %v", err)
		}
		return m, nil
	}
	logging.Warnf("Unable to fetch the P2Pool-Go manifest: %v", fetchErr)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, errManifestUnavailable
	}
	cached := &Manifest{}
	if err := json.Unmarshal(data, cached); err != nil || !cached.valid() {
		return nil, errManifestUnavailable
	}
	b, ok := cached.BinaryFor(runtime.GOOS)
	if !ok {
		return nil, errManifestUnavailable
	}
	sum, err := util.ShaSum(archiveFor(b))
	if err != nil || !strings.EqualFold(hex.EncodeToString(sum), b.Sha256) {
		return nil, errManifestUnavailable
	}
	return cached, nil
}

func writeCache(cachePath string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	tmp := cachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, cachePath); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
