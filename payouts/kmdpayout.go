package payouts

var _ Payout = &KMDPayout{}

type KMDPayout struct{}

func NewKMDPayout() *KMDPayout {
	return &KMDPayout{}
}

func (p *KMDPayout) GetID() int {
	return 14
}

func (p *KMDPayout) GetDisplayName() string {
	return "Komodo"
}

func (p *KMDPayout) GetTicker() string {
	return "KMD"
}

func (p *KMDPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *KMDPayout) GetCoingeckoCoinID() string {
	return "komodo"
}
