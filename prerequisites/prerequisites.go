package prerequisites

import (
	"github.com/vertiond/verthash-one-click-miner/logging"
)

func Install(name string) error {
	logging.Infof("Installing prerequisite [%s]\n", name)
	switch name {
	case "amddriverlinux":
		return checkAmdgpuDriverInstalled()
	case "nvidiadriverlinux":
		return checkNvidiaDriverInstalled()
	default:
		logging.Warnf("Unknown prerequisite requested: %s", name)
	}

	return nil
}
