package backend

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/miners"
	"github.com/vertcoin-project/one-click-miner-vnext/networks"
	"github.com/vertcoin-project/one-click-miner-vnext/tracking"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

func (m *Backend) PerformChecks() string {
	m.emitEvent("checkStatus", "rapidfail")
	if len(m.rapidFailures) > 0 {
		m.emitEvent("checkStatus", "Failed")
		m.rapidFailures = make([]*miners.BinaryRunner, 0) // Clear the failures
		return "Rapid failures: Your GPU is likely incompatible; check FAQ for supported hardware. If compatible, GPU overclocks or antivirus may be the cause."
	}

	m.emitEvent("checkStatus", "compatibility")
	err := m.CheckGPUCompatibility()
	if err != nil {
		tracking.Track(tracking.TrackingRequest{
			Category: "PerformChecks",
			Action:   "CheckGPUCompatibilityError",
			Name:     err.Error(),
		})
		m.emitEvent("checkStatus", "Failed")
		return err.Error()
	}

	m.emitEvent("checkStatus", "installing_miners")
	err = m.InstallMinerBinaries()
	if err != nil {
		tracking.Track(tracking.TrackingRequest{
			Category: "PerformChecks",
			Action:   "InstallMinerBinariesError",
			Name:     err.Error(),
		})
		m.emitEvent("checkStatus", "Failed")
		return err.Error()
	}

	m.emitEvent("checkStatus", "installing_p2pool")
	err = m.PrepareNode()
	if err != nil {
		trackNodeFailure("PrepareNodeError", err)
		m.emitEvent("checkStatus", "Failed")
		return err.Error()
	}

	m.emitEvent("checkStatus", "verthash")
	verthashFile := filepath.Join(util.DataDirectory(), "verthash.dat")
	err = m.checkVerthash(verthashFile)
	if err != nil {
		errorString := fmt.Sprintf("Failed to create or verify Verthash data file: %s", err.Error())
		m.emitEvent("checkStatus", "Failed")
		return errorString
	}

	m.emitEvent("checkStatus", "starting_node")
	err = m.StartNode()
	if err != nil {
		trackNodeFailure("StartNodeError", err)
		m.emitEvent("checkStatus", "Failed")
		return err.Error()
	}

	// Configuring the miners only needs the local Stratum URL, so it runs while
	// the node catches up instead of after it.
	configured := make(chan string, 1)
	go func() { configured <- m.configureMiners(m.GetArgs()) }()

	err = m.WaitNodeReady()
	if err != nil {
		<-configured
		trackNodeFailure("WaitNodeReadyError", err)
		m.emitEvent("checkStatus", "Failed")
		return err.Error()
	}

	if errorString := <-configured; errorString != "" {
		m.emitEvent("checkStatus", "Failed")
		return errorString
	}

	// The checks screen stays up until StartMining has launched the miners.
	m.emitEvent("checkStatus", "starting_miners")

	tracking.Track(tracking.TrackingRequest{
		Category: "PerformChecks",
		Action:   "Success",
	})

	return "ok"
}

// configureMiners prepares each miner for this machine's GPUs and returns a
// message for the user, or "" when every miner is ready.
func (m *Backend) configureMiners(args miners.BinaryArguments) string {
	for _, br := range m.minerBinaries {
		err := br.MinerImpl.Configure(args)
		if err != nil {
			errorString := fmt.Sprintf("Failure to configure %s: The data directory may have to be excluded from your antivirus. Check FAQ.", br.MinerBinary.MainExecutableName)
			tracking.Track(tracking.TrackingRequest{
				Category: "PerformChecks",
				Action:   "ConfigureError",
				Name:     fmt.Sprintf("%s: %v", br.MinerBinary.MainExecutableName, err),
			})
			return errorString
		}

		if br.MinerImpl.AvailableGPUs() == 0 {
			return "Miner software reported no compatible GPUs. Check FAQ for supported hardware and ensure your GPU drivers are up to date."
		}
	}
	return ""
}

func (m *Backend) CheckGPUCompatibility() error {
	gpus := util.GetGPUs()
	gpustring := ""
	for _, g := range gpus {
		if gpustring != "" {
			gpustring += " / "
		}
		gpustring += g.OSName
	}

	tracking.Track(tracking.TrackingRequest{
		Category: "EnumerateGPUs",
		Action:   "Success",
		Name:     gpustring,
	})

	return gpuCompatibility(gpus, gpustring, m.GetEnableIntegrated())
}

// gpuCompatibility fails when no GPU can mine. Intel graphics only count
// when mining on integrated graphics is turned on: otherwise the miner
// leaves them out of its config and, with nothing else, exits right away.
func gpuCompatibility(gpus []util.GPU, gpustring string, integrated bool) error {
	compat, intel := 0, 0
	for _, g := range gpus {
		switch {
		case g.Type == util.GPUTypeIntel:
			intel++
			if integrated {
				compat++
			}
		case g.Type != util.GPUTypeOther:
			compat++
		}
	}

	if compat == 0 && intel > 0 {
		return fmt.Errorf("Only integrated graphics found\n\nGPUs Found:\n%s - Mining on integrated graphics is turned off. Turn on \"Mine on integrated graphics\" in Settings to try it, but expect a very low hashrate. Check FAQ for supported hardware.", gpustring)
	}
	if compat == 0 {
		return fmt.Errorf("No compatible GPUs detected\n\nGPUs Found:\n%s - Check FAQ for supported hardware and ensure your GPU drivers are up to date.", gpustring)
	}
	return nil
}

func (m *Backend) CreateMinerBinaries() ([]*miners.BinaryRunner, error) {
	binaries := miners.GetMinerBinaries()
	gpus := util.GetGPUs()
	brs := []*miners.BinaryRunner{}
	for _, b := range binaries {
		match := false
		if b.Platform == runtime.GOOS {
			for _, g := range gpus {
				if g.Type == b.GPUType {
					match = true
				}
			}
		}

		if match {
			if b.MultiGPUMiner {
				alreadyPresent := false
				for _, br := range brs {
					if br.MinerBinary.MainExecutableName == b.MainExecutableName {
						alreadyPresent = true
						break
					}
				}
				if alreadyPresent {
					logging.Debugf("Not adding already present multi-gpu binary [%s] again\n", b.MainExecutableName)
					continue
				}
			}
			logging.Debugf("Found compatible binary [%s] for [%s/%d]\n", b.MainExecutableName, b.Platform, b.GPUType)
			br, err := miners.NewBinaryRunner(b)
			if err != nil {
				return nil, err
			}
			br.Debug = m.GetDebugging()
			brs = append(brs, br)
		} else {
			logging.Debugf("Found incompatible binary [%s] for [%s/%d]\n", b.MainExecutableName, b.Platform, b.GPUType)
		}
	}

	if len(brs) == 0 {
		return nil, fmt.Errorf("Could not find compatible miner binaries - Check FAQ for supported hardware and ensure your GPU drivers are up to date.")
	}

	return brs, nil
}

func (m *Backend) InstallMinerBinaries() error {
	var err error
	m.minerBinaries, err = m.CreateMinerBinaries()
	if err != nil {
		return err
	}

	for _, br := range m.minerBinaries {
		err := br.Install()
		if err != nil {
			return err
		}
	}
	return nil
}

// Will be run at startup
// Additionally it can be run if the backend returns an error after startup
func (m *Backend) BackendServerSelector() {
	if len(networks.Active.BackendServers) == 0 {
		networks.Active.OCMBackend = ""
		logging.Errorf("No backend servers are configured..\n")
		return
	}

	// Pick a random backend off the list
	n := rand.Intn(len(networks.Active.BackendServers))

	// Run a simple check to see if the backend is up and returned data isn't nonsense
	// If the backend is bad, go through the list until a suitable one is found
	for range networks.Active.BackendServers {
		b := util.CheckBackendStatus(networks.Active.BackendServers[n])
		if b {
			// If backend is up and return data other than 0, save it in networks.Active
			networks.Active.OCMBackend = networks.Active.BackendServers[n]
			logging.Infof("Using backend: %s\n", networks.Active.OCMBackend)
			return
		}
		n += 1
		if n == len(networks.Active.BackendServers) {
			n = 0
		}
	}
	// We'll only ever get here if all backends are unreachable..
	networks.Active.OCMBackend = networks.Active.BackendServers[0]
	logging.Errorf("No working backend could be found..\n")
}
