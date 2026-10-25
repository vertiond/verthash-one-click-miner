package util

import (
	"os"
	"path/filepath"
)

// RevealInFileManager shows path in the system file manager: the file
// selected in its folder when it exists, otherwise the nearest folder that
// does.
func RevealInFileManager(path string) {
	target, selectIt := revealTarget(path)
	revealPath(target, selectIt)
}

// revealTarget returns what to show for path and whether to select it: the
// file itself if it exists, or else its nearest existing parent folder.
func revealTarget(path string) (string, bool) {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path, true
	}
	dir := path
	for {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, false
		}
		dir = parent
	}
}

// explorerCmdLine is the raw Windows command line that opens target in
// Explorer. Explorer wants /select,"path" quoted exactly like this, which Go's
// own argument quoting gets wrong for paths with spaces.
func explorerCmdLine(target string, selectIt bool) string {
	if selectIt {
		return `explorer.exe /select,"` + target + `"`
	}
	return `explorer.exe "` + target + `"`
}
