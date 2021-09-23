package payouts

var _ Payout = &VTCPayout{}

type VTCPayout struct {}

func NewVTCPayout() *VTCPayout {
	return &VTCPayout{}
}

func (p *VTCPayout) GetID() int {
	return 1
}

func (p *VTCPayout) GetDisplayName() string {
	return "Vertcoin"
}

func (p *VTCPayout) GetTicker() string {
	return "VTC"
}

func (p *VTCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *VTCPayout) GetCoingeckoCoinID() string {
	return "vertcoin"
}
