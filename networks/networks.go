package networks

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

func SetNetwork(testnet bool) {
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
			BlockbookURLs: []string{
				"https://blockbook.doge.zelcore.io/",
				"https://dogecoin.atomicwallet.io/"},
			BackendServers: []string{
				"https://ocmbackend.javerity.com/",
				"https://ocmbackend.mindcraftblocks.com/"},
		}
	}
}
