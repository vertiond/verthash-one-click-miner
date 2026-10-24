//go:build !windows

package p2poolgo

import (
	"os"
	"path/filepath"
	"strings"
)

func coreProcessRunning() bool {
	comms, _ := filepath.Glob("/proc/*/comm")
	for _, c := range comms {
		b, err := os.ReadFile(c)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(string(b)) {
		case "vertcoind", "vertcoin-qt":
			return true
		}
	}
	return false
}
