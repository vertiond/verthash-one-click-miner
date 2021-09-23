package payouts

var _ Payout = &ZECPayout{}

type ZECPayout struct{}

func NewZECPayout() *ZECPayout {
	return &ZECPayout{}
}

func (p *ZECPayout) GetID() int {
	return 19
}

func (p *ZECPayout) GetDisplayName() string {
	return "Zcash"
}

func (p *ZECPayout) GetTicker() string {
	return "ZEC"
}

func (p *ZECPayout) GetCoingeckoExchange() string {
	return "bitfinex"
}

func (p *ZECPayout) GetCoingeckoCoinID() string {
	return "zcash"
}
