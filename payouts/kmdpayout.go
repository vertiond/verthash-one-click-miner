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

func (p *KMDPayout) GetNetworks() []string {
	return []string{}
}
