package payouts

var _ Payout = &RVCPayout{}

type RVCPayout struct{}

func NewRVCPayout() *RVCPayout {
	return &RVCPayout{}
}

func (p *RVCPayout) GetID() int {
	return 17
}

func (p *RVCPayout) GetDisplayName() string {
	return "Ravencoin Classic"
}

func (p *RVCPayout) GetTicker() string {
	return "RVC"
}

func (p *RVCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *RVCPayout) GetCoingeckoCoinID() string {
	return "ravencoin-classic"
}
