package payouts

var _ Payout = &PPCPayout{}

type PPCPayout struct{}

func NewPPCPayout() *PPCPayout {
	return &PPCPayout{}
}

func (p *PPCPayout) GetID() int {
	return 16
}

func (p *PPCPayout) GetDisplayName() string {
	return "Peercoin"
}

func (p *PPCPayout) GetTicker() string {
	return "PPC"
}

func (p *PPCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *PPCPayout) GetCoingeckoCoinID() string {
	return "peercoin"
}
