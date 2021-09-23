package payouts

var _ Payout = &ZENPayout{}

type ZENPayout struct{}

func NewZENPayout() *ZENPayout {
	return &ZENPayout{}
}

func (p *ZENPayout) GetID() int {
	return 20
}

func (p *ZENPayout) GetDisplayName() string {
	return "Horizen"
}

func (p *ZENPayout) GetTicker() string {
	return "ZEN"
}

func (p *ZENPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *ZENPayout) GetCoingeckoCoinID() string {
	return "zencash"
}
