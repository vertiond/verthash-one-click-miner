package backend

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/networks"
	"github.com/vertcoin-project/one-click-miner-vnext/p2poolgo"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
	verthash "github.com/vertcoin-project/verthash-go"
)

// nodeController is the part of *p2poolgo.Runner the backend uses; tests
// substitute a fake.
type nodeController interface {
	Config() p2poolgo.Config
	Start() error
	Stop() error
	Running() bool
	RapidFailed() bool
	LogTail(n int) []string
}

// RPCNotice tells the frontend that Vertcoin Core runs without usable RPC.
type RPCNotice struct {
	Show     bool   `json:"show"`
	ConfPath string `json:"confPath"`
	Sample   string `json:"sample"` // "server=1"
}

var errNoNodeBinary = errors.New("P2Pool-Go is not available for this platform.")

var errNodeShutdown = errors.New("One-Click Miner is shutting down.")

// Injectable for tests.
var (
	newRunner = func(cfg p2poolgo.Config) nodeController { return p2poolgo.NewRunner(cfg) }

	detectExisting = p2poolgo.DetectExisting

	// loadAndInstall installs the node binary for this platform and returns
	// its path and the manifest's peer lists.
	loadAndInstall = func() (exe string, trusted, bootstrap []string, err error) {
		man, err := p2poolgo.LoadManifest(p2poolgo.ManifestURL,
			filepath.Join(p2poolgo.BinDir(), "manifest.json"), util.GetJson, p2poolgo.ArchivePath)
		if err != nil {
			return "", nil, nil, err
		}
		b, ok := man.BinaryFor(runtime.GOOS)
		if !ok {
			return "", nil, nil, errNoNodeBinary
		}
		exe, err = p2poolgo.Install(b, p2poolgo.BinDir())
		if err != nil {
			return "", nil, nil, err
		}
		return exe, man.TrustedPeers, man.BootstrapPeers, nil
	}

	detectCore = func() p2poolgo.Detection { return p2poolgo.DetectCore(p2poolgo.ProductionCoreEnv()) }

	getStatus = p2poolgo.GetStatus

	nodeAddress = func(m *Backend) string { return m.Address() }

	nodeDataDir = func() string { return filepath.Join(util.DataDirectory(), "p2pool-go") }

	verifyVerthash = verthash.VerifyVerthashDatafile

	ensureVerthash = verthash.EnsureVerthashDatafileWithProgress

	backendDifficulty = util.GetDifficulty
	backendTipHeight  = util.GetTipHeight
	hasBackend        = func() bool { return networks.Active.OCMBackend != "" }

	nodeReadyPoll = 2 * time.Second
	nodeSlowAfter = 3 * time.Minute
)

const (
	existingNodeGap     = 3 * time.Second
	nodeStatusEvery     = 30  // mining-loop cycles (seconds)
	backendRefreshEvery = 600 // mining-loop cycles (seconds)
)

// emitEvent sends a frontend event through m.emit, or the Wails runtime.
func (m *Backend) emitEvent(name string, data ...interface{}) {
	if m.emit != nil {
		m.emit(name, data...)
		return
	}
	m.runtime.Events.Emit(name, data...)
}

// ownsRunningNode reports whether OCM's own node process is alive.
func (m *Backend) ownsRunningNode() bool {
	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	return !m.adopted && m.node != nil && m.node.Running()
}

// PrepareNode decides which node to mine to and installs ours if needed. It
// never starts a node; StartNode does.
func (m *Backend) PrepareNode() error {
	addr := nodeAddress(m)

	m.nodeMu.Lock()
	if m.nodeShutdown {
		m.nodeMu.Unlock()
		return errNodeShutdown
	}
	if !m.adopted && m.node != nil && m.node.Running() && m.node.Config().Address == addr {
		m.nodeMu.Unlock()
		return nil // healthy, nothing to install or restart
	}
	if m.node != nil {
		// Address changed, or the node is not running: stop it before any
		// reinstall, which replaces its executable.
		if err := m.node.Stop(); err != nil {
			logging.Warnf("Unable to stop P2Pool-Go: %v", err)
		}
		m.node = nil
	}
	m.adopted = false
	m.nodeNeedsStart = false
	m.nodeMu.Unlock()

	if detectExisting(p2poolgo.LocalURL(), existingNodeGap) {
		logging.Infof("Using the P2Pool-Go node already running on this computer")
		m.nodeMu.Lock()
		m.adopted = true
		m.detection = p2poolgo.Detection{}
		m.nodeMu.Unlock()
		return nil
	}

	exe, trusted, bootstrap, err := loadAndInstall()
	if err != nil {
		return err
	}
	det := detectCore()
	dataDir := nodeDataDir()
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("Unable to create the P2Pool-Go data directory: %v", err)
	}
	cfg := p2poolgo.Config{
		ExePath:        exe,
		DataDir:        dataDir,
		VerthashPath:   filepath.Join(util.DataDirectory(), "verthash.dat"),
		Address:        addr,
		Detection:      det,
		TrustedPeers:   trusted,
		BootstrapPeers: bootstrap,
	}

	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	if m.nodeShutdown {
		return errNodeShutdown
	}
	m.detection = det
	m.node = newRunner(cfg)
	m.nodeNeedsStart = true
	return nil
}

// StartNode starts the runner PrepareNode (or the verthash step) built. It
// is a no-op for a reused or adopted node.
func (m *Backend) StartNode() error {
	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	if m.nodeShutdown {
		return errNodeShutdown
	}
	if m.adopted || !m.nodeNeedsStart {
		return nil
	}
	if m.node == nil {
		return errors.New("P2Pool-Go was not prepared")
	}
	if err := m.node.Start(); err != nil {
		return err
	}
	m.nodeNeedsStart = false
	return nil
}

// checkVerthash makes sure verthash.dat is valid without rewriting it under
// a running node: with our node running it only verifies, and on failure it
// stops the node, regenerates, and has StartNode start a fresh runner.
func (m *Backend) checkVerthash(file string) error {
	if m.ownsRunningNode() {
		ok, err := verifyVerthash(file)
		if err == nil && ok {
			return nil
		}
		logging.Warnf("Verthash data file invalid while P2Pool-Go runs (ok=%v, err=%v); stopping the node to regenerate it", ok, err)
		m.nodeMu.Lock()
		if m.node != nil {
			cfg := m.node.Config()
			if err := m.node.Stop(); err != nil {
				logging.Warnf("Unable to stop P2Pool-Go: %v", err)
			}
			m.node = newRunner(cfg)
			m.nodeNeedsStart = true
		}
		m.nodeMu.Unlock()
	}

	doneChan := make(chan error, 1)
	progress := make(chan float64, 1)
	go func() {
		doneChan <- ensureVerthash(file, progress)
	}()
	for {
		select {
		case err := <-doneChan:
			return err
		case prog := <-progress:
			m.emitEvent("verthashProgress", prog*100)
		}
	}
}

// WaitNodeReady polls the node until it is ready, emitting checkStatus
// node_<state>. It gives up only on a node rapid-fail or shutdown.
func (m *Backend) WaitNodeReady() error {
	select { // drop a Retry pressed before this wait began
	case <-m.nodeRetry:
	default:
	}
	since := time.Now()
	slow := false
	defer func() {
		if slow {
			m.emitEvent("nodeSlow", false)
		}
	}()

	for {
		m.nodeMu.Lock()
		node, adopted, shut := m.node, m.adopted, m.nodeShutdown
		m.nodeMu.Unlock()
		if shut {
			return errNodeShutdown
		}
		if !adopted && node == nil {
			return errors.New("P2Pool-Go was not prepared")
		}
		if !adopted && node.RapidFailed() {
			return fmt.Errorf("P2Pool-Go stopped repeatedly. Last lines of p2pool.log:\n%s",
				strings.Join(node.LogTail(20), "\n"))
		}

		st, err := getStatus(p2poolgo.LocalURL(), nodeAddress(m))
		switch {
		case err == nil && st.Ready:
			m.emitEvent("checkStatus", "node_ready")
			return nil
		case err == nil:
			m.emitEvent("checkStatus", "node_"+st.State)
		default:
			// Not answering yet: /api/status comes up after verthash, the
			// sharechain load and P2P start.
			m.emitEvent("checkStatus", "node_starting")
		}

		if !slow && time.Since(since) >= nodeSlowAfter {
			slow = true
			m.emitEvent("nodeSlow", true)
		}

		select {
		case <-m.nodeRetry:
			if err := m.restartNode(); err != nil {
				return err
			}
			since = time.Now()
			if slow {
				slow = false
				m.emitEvent("nodeSlow", false)
			}
		case <-time.After(nodeReadyPoll):
		}
	}
}

// restartNode stops and starts our node, or re-runs detection when adopted.
func (m *Backend) restartNode() error {
	m.nodeMu.Lock()
	if m.nodeShutdown {
		m.nodeMu.Unlock()
		return errNodeShutdown
	}
	if !m.adopted && m.node != nil {
		defer m.nodeMu.Unlock()
		logging.Infof("Restarting P2Pool-Go on request")
		if err := m.node.Stop(); err != nil {
			logging.Warnf("Unable to stop P2Pool-Go: %v", err)
		}
		m.nodeNeedsStart = false
		return m.node.Start()
	}
	m.adopted = false
	m.nodeMu.Unlock()
	if err := m.PrepareNode(); err != nil {
		return err
	}
	return m.StartNode()
}

// RetryNode asks WaitNodeReady to restart the node. Bound to the frontend.
func (m *Backend) RetryNode() {
	select {
	case m.nodeRetry <- true:
	default:
	}
}

// GetRPCNotice reports whether to tell the user to enable Core's RPC.
// Bound to the frontend.
func (m *Backend) GetRPCNotice() RPCNotice {
	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	if m.adopted || !m.detection.RPCNotice {
		return RPCNotice{}
	}
	return RPCNotice{Show: true, ConfPath: m.detection.ConfPath, Sample: "server=1"}
}

// ShutdownNode stops our node on quit and keeps it from being started
// again. An adopted node is left running.
func (m *Backend) ShutdownNode() {
	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	m.nodeShutdown = true
	if m.adopted || m.node == nil {
		return
	}
	logging.Infof("Stopping P2Pool-Go...")
	if err := m.node.Stop(); err != nil {
		logging.Errorf("Error stopping P2Pool-Go: %v", err)
	}
}

// networkState carries the mining loop's cached network numbers.
type networkState struct {
	node         *p2poolgo.Status
	backendTip   int64
	backendStale bool
}

// networkNumbers returns the difficulty and tip height for the mining screen
// at loop cycle n (one per second): the node is polled every 30 cycles, the
// backend every 600 (and as soon as it is known, if it was not yet).
func (m *Backend) networkNumbers(cycle int, ns *networkState) (float64, int64) {
	if cycle%nodeStatusEvery == 0 {
		st, err := getStatus(p2poolgo.LocalURL(), nodeAddress(m))
		if err != nil {
			st = nil
		}
		ns.node = st
	}
	if cycle%backendRefreshEvery == 0 {
		ns.backendStale = true
	}
	if ns.backendStale && hasBackend() {
		ns.backendStale = false
		if d := backendDifficulty(); d > 0 {
			m.lastBackendDiff = d
		}
		if t := backendTipHeight(); t > 0 {
			ns.backendTip = t
		}
	}
	diff, tip, _ := p2poolgo.ChooseNetwork(ns.node, m.lastBackendDiff, ns.backendTip)
	return diff, tip
}
