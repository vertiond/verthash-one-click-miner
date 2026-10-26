package networks

type Network struct {
	Base58P2PKHVersion byte
	Base58P2SHVersion  byte
	InsightURL         string
	Bech32Prefix       string
	WalletDB           string
	OCMBackend         string
	BackendServers     []string
}

var Active Network

func SetNetwork() {
	Active = Network{
		Base58P2PKHVersion: 71,
		Base58P2SHVersion:  5,
		InsightURL:         "https://insight.vertcoin.org/",
		Bech32Prefix:       "vtc",
		WalletDB:           "wallet-testnet.db", // The mainnet wallet has always used this name; renaming it would hide existing wallets
		BackendServers: []string{
			"https://ocmbackend.javerity.com/",
			"https://ocmbackend.mindcraftblocks.com/"},
	}
}
