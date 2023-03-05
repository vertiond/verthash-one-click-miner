package payouts

var _ Payout = &BNBPayout{}

type BNBPayout struct{}

func NewBNBPayout() *BNBPayout {
	return &BNBPayout{}
}

func (p *BNBPayout) GetID() int {
	return 27
}

func (p *BNBPayout) GetDisplayName() string {
	return "Binance Coin"
}

func (p *BNBPayout) GetTicker() string {
	return "BNB"
}

func (p *BNBPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *BNBPayout) GetCoingeckoCoinID() string {
	return "binancecoin"
}

func (p *BNBPayout) GetNetworks() []string {
	return []string{}
}
