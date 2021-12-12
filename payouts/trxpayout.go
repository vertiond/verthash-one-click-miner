package payouts

var _ Payout = &TRXPayout{}

type TRXPayout struct{}

func NewTRXPayout() *TRXPayout {
	return &TRXPayout{}
}

func (p *TRXPayout) GetID() int {
	return 26
}

func (p *TRXPayout) GetDisplayName() string {
	return "Tron"
}

func (p *TRXPayout) GetTicker() string {
	return "TRX"
}

func (p *TRXPayout) GetCoingeckoExchange() string {
	return "binance"
}

func (p *TRXPayout) GetCoingeckoCoinID() string {
	return "tron"
}

func (p *TRXPayout) GetNetworks() []string {
	return []string{}
}
