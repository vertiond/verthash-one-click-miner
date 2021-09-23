package payouts

var _ Payout = &DGBPayout{}

type DGBPayout struct{}

func NewDGBPayout() *DGBPayout {
	return &DGBPayout{}
}

func (p *DGBPayout) GetID() int {
	return 9
}

func (p *DGBPayout) GetDisplayName() string {
	return "Digibyte"
}

func (p *DGBPayout) GetTicker() string {
	return "DGB"
}

func (p *DGBPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *DGBPayout) GetCoingeckoCoinID() string {
	return "digibyte"
}
