package p2poolgo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vertcoin-project/one-click-miner-vnext/miners"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

// BinDir holds the downloaded node archives and the unpacked "current" copy.
// Nothing stateful lives here (node state is in util.DataDirectory()/p2pool-go).
func BinDir() string {
	return filepath.Join(util.DataDirectory(), "p2pool-go-bin")
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return "p2pool-go-vtc.exe"
	}
	return "p2pool-go-vtc"
}

func archivePathIn(binDir string, b Binary) string {
	return filepath.Join(binDir, b.Sha256+".zip")
}

func exePathIn(binDir string) string {
	return filepath.Join(binDir, "current", exeName())
}

// ArchivePath is where the archive for b is stored.
func ArchivePath(b Binary) string { return archivePathIn(BinDir(), b) }

// ExePath is the fixed path of the node executable. It does not change across
// versions, which keeps the Windows Firewall rule stable.
func ExePath() string { return exePathIn(BinDir()) }

// Install downloads (if needed) and verifies the archive for b, then unpacks
// it into binDir/current, replacing any previous copy. Callers must not call
// it while OCM's own node is running.
func Install(b Binary, binDir string) (string, error) {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}
	archive := archivePathIn(binDir, b)
	if err := miners.EnsureArchive(b.Url, b.Sha256, archive); err != nil {
		return "", err
	}
	if err := miners.UnpackArchive(b.Url, archive, filepath.Join(binDir, "current")); err != nil {
		return "", err
	}
	exe := exePathIn(binDir)
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("The P2Pool-Go archive does not contain %s", exeName())
	}
	if _, err := os.Stat(filepath.Join(binDir, "current", "web", "static")); err != nil {
		return "", fmt.Errorf("The P2Pool-Go archive does not contain web/static")
	}
	return exe, nil
}
