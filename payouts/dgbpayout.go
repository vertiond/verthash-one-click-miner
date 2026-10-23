package payouts

var _ Payout = &DGBPayout{}

type DGBPayout struct{}

func NewDGBPayout() *DGBPayout {
	return &DGBPayout{}
}

func (p *DGBPayout) GetID() int {
	return 9
}

func (p *DGBPayout) GetDisplayName() string {
	return "Digibyte"
}

func (p *DGBPayout) GetTicker() string {
	return "DGB"
}

func (p *DGBPayout) GetNetworks() []string {
	return []string{}
}
