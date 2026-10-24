//go:build windows

package p2poolgo

import (
	"os/exec"
	"strings"

	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

func coreProcessRunning() bool {
	for _, name := range []string{"vertcoind.exe", "vertcoin-qt.exe"} {
		cmd := exec.Command("tasklist", "/NH", "/FI", "IMAGENAME eq "+name)
		util.PrepareBackgroundCommand(cmd)
		out, err := cmd.Output()
		if err == nil && strings.Contains(strings.ToLower(string(out)), name) {
			return true
		}
	}
	return false
}
