package networks

import "sync"

type Network struct {
	Base58P2PKHVersion byte
	Base58P2SHVersion  byte
	ExplorerTxURL      string
	Bech32Prefix       string
	P2ProxyStratum     string
	P2ProxyURL         string
	WalletDB           string
	OCMBackend         string
	BackendServers     []string
	BlockbookURLs      []string
}

var Active Network

// DefaultBlockbookURLs is the mainnet Blockbook list used when neither
// backends.json nor its cached copy has one.
var DefaultBlockbookURLs = []string{
	"https://blockbook.doge.zelcore.io/",
	"https://dogecoin.atomicwallet.io/"}

var (
	networkMu            sync.Mutex
	mainnetBlockbookURLs = DefaultBlockbookURLs
	testnetActive        bool
	endpointsResolved    = closedChan()
)

func closedChan() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

// HoldEndpoints makes WaitForEndpoints block until the returned release
// function is called. Call it before anything can use the endpoints.
func HoldEndpoints() (release func()) {
	c := make(chan struct{})
	networkMu.Lock()
	endpointsResolved = c
	networkMu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { close(c) }) }
}

// WaitForEndpoints blocks until the remote endpoints have been resolved. It
// returns at once when HoldEndpoints was never called.
func WaitForEndpoints() {
	networkMu.Lock()
	c := endpointsResolved
	networkMu.Unlock()
	<-c
}

// SetBlockbookURLs replaces the mainnet Blockbook list. It also applies to
// Active when the mainnet is selected. An empty list is ignored.
func SetBlockbookURLs(urls []string) {
	if len(urls) == 0 {
		return
	}
	networkMu.Lock()
	defer networkMu.Unlock()
	mainnetBlockbookURLs = urls
	if !testnetActive {
		Active.BlockbookURLs = urls
	}
}

// DefaultBackendServers is the mainnet backend list used when neither
// backends.json nor its cached copy is available.
var DefaultBackendServers = []string{
	"https://ocmbackend.javerity.com/",
	"https://ocmbackend.mindcraftblocks.com/"}

var mainnetBackendServers = DefaultBackendServers

// SetBackendServers replaces the mainnet backend list. It also applies to
// Active when the mainnet is selected.
func SetBackendServers(servers []string) {
	networkMu.Lock()
	defer networkMu.Unlock()
	mainnetBackendServers = servers
	if !testnetActive {
		Active.BackendServers = servers
	}
}

func SetNetwork(testnet bool) {
	networkMu.Lock()
	defer networkMu.Unlock()
	testnetActive = testnet
	if testnet {
		Active = Network{
			Base58P2PKHVersion: 74,
			Base58P2SHVersion:  196,
			ExplorerTxURL:      "https://dogechain.com/tx/",
			Bech32Prefix:       "tvtc",
			P2ProxyStratum:     "stratum+tcp://p2proxy-testnet.gertjaap.org:9171",
			P2ProxyURL:         "https://p2proxy-testnet.gertjaap.org/",
			WalletDB:           "wallet-testnet.db",
		}
	} else {
		Active = Network{
			Base58P2PKHVersion: 30,
			Base58P2SHVersion:  22,
			ExplorerTxURL:      "https://dogechain.com/tx/",
			Bech32Prefix:       "vtc",
			P2ProxyStratum:     "stratum+tcp://p2proxy.vertcoin.org:9171",
			P2ProxyURL:         "https://p2proxy.vertcoin.org/",
			WalletDB:           "wallet-testnet.db",
			BlockbookURLs:      mainnetBlockbookURLs,
			BackendServers:     mainnetBackendServers,
		}
	}
}
