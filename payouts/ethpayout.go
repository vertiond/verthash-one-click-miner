package payouts

var _ Payout = &ETHPayout{}

type ETHPayout struct{}

func NewETHPayout() *ETHPayout {
	return &ETHPayout{}
}

func (p *ETHPayout) GetID() int {
	return 7
}

func (p *ETHPayout) GetDisplayName() string {
	return "Ethereum"
}

func (p *ETHPayout) GetTicker() string {
	return "ETH"
}

func (p *ETHPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *ETHPayout) GetCoingeckoCoinID() string {
	return "ethereum"
}
