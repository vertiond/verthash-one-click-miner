package payouts

var _ Payout = &DOGEPayout{}

type DOGEPayout struct{}

func NewDOGEPayout() *DOGEPayout {
	return &DOGEPayout{}
}

func (p *DOGEPayout) GetID() int {
	return 4
}

func (p *DOGEPayout) GetDisplayName() string {
	return "Verthash OCM Dogecoin Wallet"
}

func (p *DOGEPayout) GetTicker() string {
	return "DOGE"
}

func (p *DOGEPayout) GetCoingeckoExchange() string {
	return "bittrex"
}

func (p *DOGEPayout) GetCoingeckoCoinID() string {
	return "dogecoin"
}
