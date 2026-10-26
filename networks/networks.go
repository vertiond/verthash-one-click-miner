package networks

import "sync"

type Network struct {
	Base58P2PKHVersion byte
	Base58P2SHVersion  byte
	ExplorerTxURL      string
	Bech32Prefix       string
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

// SetBlockbookURLs replaces the mainnet Blockbook list and applies it to
// Active. An empty list is ignored.
func SetBlockbookURLs(urls []string) {
	if len(urls) == 0 {
		return
	}
	networkMu.Lock()
	defer networkMu.Unlock()
	mainnetBlockbookURLs = urls
	Active.BlockbookURLs = urls
}

// DefaultBackendServers is the mainnet backend list used when neither
// backends.json nor its cached copy is available.
var DefaultBackendServers = []string{
	"https://ocmbackend.javerity.com/",
	"https://ocmbackend.mindcraftblocks.com/"}

var mainnetBackendServers = DefaultBackendServers

// SetBackendServers replaces the mainnet backend list and applies it to
// Active.
func SetBackendServers(servers []string) {
	networkMu.Lock()
	defer networkMu.Unlock()
	mainnetBackendServers = servers
	Active.BackendServers = servers
}

func SetNetwork() {
	networkMu.Lock()
	defer networkMu.Unlock()
	Active = Network{
		Base58P2PKHVersion: 30,
		Base58P2SHVersion:  22,
		ExplorerTxURL:      "https://dogechain.com/tx/",
		Bech32Prefix:       "vtc",
		WalletDB:           "wallet-testnet.db", // The mainnet wallet has always used this name; renaming it would hide existing wallets
		BlockbookURLs:      mainnetBlockbookURLs,
		BackendServers:     mainnetBackendServers,
	}
}
