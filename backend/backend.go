package backend

import (
	"path/filepath"
	"sync"

	"github.com/tidwall/buntdb"

	"github.com/btcsuite/btcd/wire"
	"github.com/vertcoin-project/one-click-miner-vnext/miners"
	"github.com/vertcoin-project/one-click-miner-vnext/p2poolgo"
	"github.com/vertcoin-project/one-click-miner-vnext/pools"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
	"github.com/vertcoin-project/one-click-miner-vnext/wallet"
	"github.com/wailsapp/wails"
)

type Backend struct {
	runtime             *wails.Runtime
	wal                 *wallet.Wallet
	settings            *buntdb.DB
	pendingSweep        []*wire.MsgTx
	minerBinaries       []*miners.BinaryRunner
	rapidFailures       []*miners.BinaryRunner
	pool                pools.Pool
	refreshBalanceChan  chan bool
	refreshHashChan     chan bool
	refreshRunningState chan bool
	stopMonitoring      chan bool
	stopHash            chan bool
	stopBalance         chan bool
	stopUpdate          chan bool
	stopRunningState    chan bool
	prerequisiteInstall chan bool
	alreadyRunning      bool

	// emit sends frontend events; nil means m.runtime.Events.Emit.
	emit func(string, ...interface{})

	// The P2Pool-Go node. nodeMu guards these fields; only PerformChecks and
	// ShutdownNode start or stop the node.
	nodeMu         sync.Mutex
	node           nodeController
	adopted        bool // mining to a node OCM did not start
	detection      p2poolgo.Detection
	nodeNeedsStart bool // node was built (or rebuilt) and StartNode must start it
	nodeShutdown   bool // OCM is quitting; never start a node again
	nodeRetry      chan bool

	lastBackendDiff float64 // last difficulty from the backend /info; 0 = never reached
}

func NewBackend(alreadyRunning bool) (*Backend, error) {
	backend := &Backend{
		refreshBalanceChan:  make(chan bool),
		refreshHashChan:     make(chan bool),
		refreshRunningState: make(chan bool),
		stopHash:            make(chan bool),
		stopBalance:         make(chan bool),
		stopRunningState:    make(chan bool),
		stopMonitoring:      make(chan bool),
		stopUpdate:          make(chan bool),
		prerequisiteInstall: make(chan bool),
		nodeRetry:           make(chan bool, 1),
		minerBinaries:       []*miners.BinaryRunner{},
		rapidFailures:       []*miners.BinaryRunner{},
	}

	if alreadyRunning {
		backend.alreadyRunning = true
		return backend, nil
	}

	db, err := buntdb.Open(filepath.Join(util.DataDirectory(), "settings.db"))
	if err != nil {
		return nil, err
	}
	backend.settings = db
	return backend, nil
}

func (m *Backend) ResetPool() {
	m.pool = pools.NewP2PoolGo(m.Address())
}

func (m *Backend) WailsInit(runtime *wails.Runtime) error {
	// Save runtime
	m.runtime = runtime

	go m.PrerequisiteProxyLoop()
	go m.UpdateLoop()

	return nil
}

func (m *Backend) OpenDownloadUrl(url string) {
	util.OpenBrowser(url)
}

func (m *Backend) AlreadyRunning() bool {
	return m.alreadyRunning
}

func (m *Backend) Close() {
	m.runtime.Window.Close()
}
