package payouts

var _ Payout = &ETCPayout{}

type ETCPayout struct{}

func NewETCPayout() *ETCPayout {
	return &ETCPayout{}
}

func (p *ETCPayout) GetID() int {
	return 21
}

func (p *ETCPayout) GetDisplayName() string {
	return "Ethereum Classic"
}

func (p *ETCPayout) GetTicker() string {
	return "ETC"
}

func (p *ETCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *ETCPayout) GetCoingeckoCoinID() string {
	return "ethereum-classic"
}
