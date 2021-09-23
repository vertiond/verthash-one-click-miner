package payouts

var _ Payout = &BCHPayout{}

type BCHPayout struct{}

func NewBCHPayout() *BCHPayout {
	return &BCHPayout{}
}

func (p *BCHPayout) GetID() int {
	return 5
}

func (p *BCHPayout) GetDisplayName() string {
	return "Bitcoin Cash"
}

func (p *BCHPayout) GetTicker() string {
	return "BCH"
}

func (p *BCHPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *BCHPayout) GetCoingeckoCoinID() string {
	return "bitcoin-cash"
}
