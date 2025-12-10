package payouts

var _ Payout = &DOGEPayout{}

type DOGEPayout struct{}

func NewDOGEPayout() *DOGEPayout {
	return &DOGEPayout{}
}

func (p *DOGEPayout) GetID() int {
	return 4
}

func (p *DOGEPayout) GetDisplayName() string {
	return "Verthash OCM Dogecoin Wallet"
}

func (p *DOGEPayout) GetTicker() string {
	// Returns "DOGE" - BTC will be appended in GetBitcoinPerUnitCoin to form "DOGEBTC" for CoinEx API
	return "DOGE"
}

func (p *DOGEPayout) GetCoingeckoExchange() string {
	// Returns "binance" which is used as the exchange parameter for freecryptoapi.com API
	return "binance"
}

func (p *DOGEPayout) GetCoingeckoCoinID() string {
	return "dogecoin"
}

func (p *DOGEPayout) GetNetworks() []string {
	return []string{}
}
