package payouts

var _ Payout = &RTMPayout{}

type RTMPayout struct{}

func NewRTMPayout() *RTMPayout {
	return &RTMPayout{}
}

func (p *RTMPayout) GetID() int {
	return 22
}

func (p *RTMPayout) GetDisplayName() string {
	return "Raptoreum"
}

func (p *RTMPayout) GetTicker() string {
	return "RTM"
}

func (p *RTMPayout) GetCoingeckoExchange() string {
	return "coinex"
}

func (p *RTMPayout) GetCoingeckoCoinID() string {
	return "raptoreum"
}
