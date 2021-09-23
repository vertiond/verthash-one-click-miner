package payouts

var _ Payout = &BTCPayout{}

type BTCPayout struct {}

func NewBTCPayout() *BTCPayout {
	return &BTCPayout{}
}

func (p *BTCPayout) GetID() int {
	return 2
}

func (p *BTCPayout) GetDisplayName() string {
	return "Bitcoin"
}

func (p *BTCPayout) GetTicker() string {
	return "BTC"
}

func (p *BTCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *BTCPayout) GetCoingeckoCoinID() string {
	return "bitcoin"
}
