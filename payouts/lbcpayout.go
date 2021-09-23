package payouts

var _ Payout = &LBCPayout{}

type LBCPayout struct{}

func NewLBCPayout() *LBCPayout {
	return &LBCPayout{}
}

func (p *LBCPayout) GetID() int {
	return 15
}

func (p *LBCPayout) GetDisplayName() string {
	return "LBRY Credits"
}

func (p *LBCPayout) GetTicker() string {
	return "LBC"
}

func (p *LBCPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *LBCPayout) GetCoingeckoCoinID() string {
	return "lbry-credits"
}
