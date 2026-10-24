package miners

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

// EnsureArchive makes sure the file at path exists and has the given SHA256
// (hex). It downloads url if the file is missing. If the existing file has a
// different hash it is deleted and downloaded once more; a fresh download
// that still mismatches is an error.
func EnsureArchive(url, sha256Hex, path string) error {
	freshDownload := false
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		logging.Debugf("Archive not found, downloading...")
		freshDownload = true
		if err := downloadArchive(url, path); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		logging.Debugf("Archive file already exists")
	}

	shaSum, err := util.ShaSum(path)
	if err != nil {
		return err
	}
	expectedHash, _ := hex.DecodeString(sha256Hex)
	if !bytes.Equal(shaSum, expectedHash) {
		logging.Warnf("Hash differs: [%x] vs [%s]", shaSum, sha256Hex)
		if !freshDownload {
			if err := os.Remove(path); err != nil {
				return err
			}
			return EnsureArchive(url, sha256Hex, path)
		}
		err = fmt.Errorf("Freshly downloaded archive did not have correct SHA256 hash")
		logging.Error(err)
		return err
	}

	logging.Debugf("Archive file is available and correct")
	return nil
}

func downloadArchive(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	logging.Debugf("Archive file downloaded")
	return err
}

// UnpackArchive makes destDir hold exactly the files of archivePath. The
// archive format (zip or tar.gz) is chosen by the suffix of url.
//
// Syncing instead of deleting and unpacking again leaves files that are still
// right untouched, so a process from the last run, or an antivirus scan,
// still holding the executable open doesn't make this fail.
func UnpackArchive(url, archivePath, destDir string) error {
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return err
	}

	logging.Debugf("Syncing unpack directory with the archive")
	if strings.HasSuffix(url, ".zip") {
		return util.SyncZip(archivePath, destDir)
	} else if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		return util.SyncTar(archivePath, destDir)
	}

	return fmt.Errorf("Unknown archive format, cannot unpack: %s", url)
}
