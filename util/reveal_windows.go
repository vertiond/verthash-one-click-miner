//go:build windows
// +build windows

package util

import (
	"os/exec"
	"syscall"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
)

func revealPath(target string, selectIt bool) {
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: explorerCmdLine(target, selectIt)}
	// Explorer's exit code means nothing (it is 1 even when it worked), so only
	// a failure to start is worth logging.
	if err := cmd.Start(); err != nil {
		logging.Errorf("Unable to open %s in Explorer: %v", target, err)
		return
	}
	go func() { _ = cmd.Wait() }()
}
