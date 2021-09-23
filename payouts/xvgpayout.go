package payouts

var _ Payout = &XVGPayout{}

type XVGPayout struct{}

func NewXVGPayout() *XVGPayout {
	return &XVGPayout{}
}

func (p *XVGPayout) GetID() int {
	return 18
}

func (p *XVGPayout) GetDisplayName() string {
	return "Verge"
}

func (p *XVGPayout) GetTicker() string {
	return "XVG"
}

func (p *XVGPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *XVGPayout) GetCoingeckoCoinID() string {
	return "verge"
}
