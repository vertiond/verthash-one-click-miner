
package payouts

var _ Payout = &USDTPayout{}

type USDTPayout struct{}

func NewUSDTPayout() *USDTPayout {
	return &USDTPayout{}
}

func (p *USDTPayout) GetID() int {
	return 23
}

func (p *USDTPayout) GetDisplayName() string {
	return "Tether"
}

func (p *USDTPayout) GetTicker() string {
	return "USDT"
}

func (p *USDTPayout) GetNetworks() []string {
	return []string{
		"ERC20",
		"TRC20",
		"BEP20",
	}
}
