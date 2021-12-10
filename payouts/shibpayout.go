
package payouts

var _ Payout = &SHIBPayout{}

type SHIBPayout struct{}

func NewSHIBPayout() *SHIBPayout {
	return &SHIBPayout{}
}

func (p *SHIBPayout) GetID() int {
	return 24
}

func (p *SHIBPayout) GetDisplayName() string {
	return "Shiba Inu"
}

func (p *SHIBPayout) GetTicker() string {
	return "SHIB"
}

func (p *SHIBPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *SHIBPayout) GetCoingeckoCoinID() string {
	return "shiba-inu"
}
