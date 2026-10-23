package payouts

var _ Payout = &RVNPayout{}

type RVNPayout struct{}

func NewRVNPayout() *RVNPayout {
	return &RVNPayout{}
}

func (p *RVNPayout) GetID() int {
	return 8
}

func (p *RVNPayout) GetDisplayName() string {
	return "Ravencoin"
}

func (p *RVNPayout) GetTicker() string {
	return "RVN"
}

func (p *RVNPayout) GetNetworks() []string {
	return []string{}
}
