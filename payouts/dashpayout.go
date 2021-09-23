package payouts

var _ Payout = &DASHPayout{}

type DASHPayout struct{}

func NewDASHPayout() *DASHPayout {
	return &DASHPayout{}
}

func (p *DASHPayout) GetID() int {
	return 6
}

func (p *DASHPayout) GetDisplayName() string {
	return "Dash"
}

func (p *DASHPayout) GetTicker() string {
	return "DASH"
}

func (p *DASHPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *DASHPayout) GetCoingeckoCoinID() string {
	return "dash"
}
