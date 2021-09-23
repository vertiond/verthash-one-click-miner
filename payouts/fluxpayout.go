package payouts

var _ Payout = &FLUXPayout{}

type FLUXPayout struct{}

func NewFLUXPayout() *FLUXPayout {
	return &FLUXPayout{}
}

func (p *FLUXPayout) GetID() int {
	return 13
}

func (p *FLUXPayout) GetDisplayName() string {
	return "Flux"
}

func (p *FLUXPayout) GetTicker() string {
	return "FLUX"
}

func (p *FLUXPayout) GetCoingeckoExchange() string {
	return "coinex"
}

func (p *FLUXPayout) GetCoingeckoCoinID() string {
	return "zelcash"
}
