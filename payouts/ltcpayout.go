package payouts

var _ Payout = &LTCPayout{}

type LTCPayout struct {}

func NewLTCPayout() *LTCPayout {
	return &LTCPayout{}
}

func (p *LTCPayout) GetID() int {
	return 3
}

func (p *LTCPayout) GetDisplayName() string {
	return "Litecoin"
}

func (p *LTCPayout) GetTicker() string {
	return "LTC"
}

func (p *LTCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *LTCPayout) GetCoingeckoCoinID() string {
	return "litecoin"
}
