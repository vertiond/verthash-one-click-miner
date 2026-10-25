//go:build !windows
// +build !windows

package util

import (
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
)

func revealPath(target string, selectIt bool) {
	var cmd *exec.Cmd
	switch {
	case runtime.GOOS == "darwin" && selectIt:
		cmd = exec.Command("open", "-R", target)
	case runtime.GOOS == "darwin":
		cmd = exec.Command("open", target)
	default:
		// xdg-open can't select a file, so open the folder that holds it.
		if selectIt {
			target = filepath.Dir(target)
		}
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		logging.Errorf("Unable to open %s in the file manager: %v", target, err)
		return
	}
	go func() { _ = cmd.Wait() }()
}
