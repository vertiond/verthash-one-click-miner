package payouts

var _ Payout = &XMRPayout{}

type XMRPayout struct{}

func NewXMRPayout() *XMRPayout {
	return &XMRPayout{}
}

func (p *XMRPayout) GetID() int {
	return 12
}

func (p *XMRPayout) GetDisplayName() string {
	return "Monero"
}

func (p *XMRPayout) GetTicker() string {
	return "XMR"
}

func (p *XMRPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *XMRPayout) GetCoingeckoCoinID() string {
	return "monero"
}
