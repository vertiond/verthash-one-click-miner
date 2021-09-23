package payouts

var _ Payout = &FIROPayout{}

type FIROPayout struct{}

func NewFIROPayout() *FIROPayout {
	return &FIROPayout{}
}

func (p *FIROPayout) GetID() int {
	return 11
}

func (p *FIROPayout) GetDisplayName() string {
	return "Firo"
}

func (p *FIROPayout) GetTicker() string {
	return "FIRO"
}

func (p *FIROPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *FIROPayout) GetCoingeckoCoinID() string {
	return "zcoin"
}
