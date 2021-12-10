package payouts

var _ Payout = &MATICPayout{}

type MATICPayout struct{}

func NewMATICPayout() *MATICPayout {
	return &MATICPayout{}
}

func (p *MATICPayout) GetID() int {
	return 25
}

func (p *MATICPayout) GetDisplayName() string {
	return "Polygon"
}

func (p *MATICPayout) GetTicker() string {
	return "MATIC"
}

func (p *MATICPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *MATICPayout) GetCoingeckoCoinID() string {
	return "matic"
}
