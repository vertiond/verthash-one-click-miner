package payouts

var _ Payout = &BTRMPayout{}

type BTRMPayout struct {}

func NewBTRMPayout() *BTRMPayout {
	return &BTRMPayout{}
}

func (p *BTRMPayout) GetID() int {
	return 29
}

func (p *BTRMPayout) GetDisplayName() string {
	return "Bitoreum"
}

func (p *BTRMPayout) GetTicker() string {
	return "BTRM"
}

func (p *BTRMPayout) GetCoingeckoExchange() string {
	return "Txbit"
}

func (p *BTRMPayout) GetCoingeckoCoinID() string {
	return "Bitoreum"
}

func (p *BTRMPayout) GetNetworks() []string {
	return []string{}
}
