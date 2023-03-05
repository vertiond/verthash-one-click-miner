package payouts

var _ Payout = &CAKEPayout{}

type CAKEPayout struct{}

func NewCAKEPayout() *CAKEPayout {
	return &CAKEPayout{}
}

func (p *CAKEPayout) GetID() int {
	return 28
}

func (p *CAKEPayout) GetDisplayName() string {
	return "Cake"
}

func (p *CAKEPayout) GetTicker() string {
	return "CAKE"
}

func (p *CAKEPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *CAKEPayout) GetCoingeckoCoinID() string {
	return "pancakeswap-token"
}

func (p *CAKEPayout) GetNetworks() []string {
	return []string{}
}
