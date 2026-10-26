package payouts

import (
	"sync"
)

// IDs the app relies on. They are saved in the settings, so pools.json must
// keep them.
const (
	VTCID  = 1
	DOGEID = 4
)

// Config describes one payout currency in pools.json. Networks lists the
// chain networks the currency can be paid on, if the pool asks for one.
type Config struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Ticker   string   `json:"ticker"`
	Networks []string `json:"networks,omitempty"`
}

var _ Payout = &ConfiguredPayout{}

// ConfiguredPayout is a payout currency described by a pools.json entry.
type ConfiguredPayout struct {
	Config Config
}

func NewConfiguredPayout(c Config) *ConfiguredPayout {
	return &ConfiguredPayout{Config: c}
}

func (p *ConfiguredPayout) GetID() int {
	return p.Config.ID
}

func (p *ConfiguredPayout) GetDisplayName() string {
	return p.Config.Name
}

func (p *ConfiguredPayout) GetTicker() string {
	return p.Config.Ticker
}

func (p *ConfiguredPayout) GetNetworks() []string {
	return append([]string{}, p.Config.Networks...)
}

// DefaultConfigs is the payout list used when neither pools.json nor its
// cached copy is available.
var DefaultConfigs = []Config{
	{ID: 1, Name: "Vertcoin", Ticker: "VTC"},
	{ID: 2, Name: "Bitcoin", Ticker: "BTC"},
	{ID: 3, Name: "Litecoin", Ticker: "LTC"},
	{ID: 4, Name: "Verthash OCM Dogecoin Wallet", Ticker: "DOGE"},
	{ID: 5, Name: "Bitcoin Cash", Ticker: "BCH"},
	{ID: 6, Name: "Dash", Ticker: "DASH"},
	{ID: 8, Name: "Ravencoin", Ticker: "RVN"},
	{ID: 9, Name: "Digibyte", Ticker: "DGB"},
	{ID: 10, Name: "Groestlcoin", Ticker: "GRS"},
	{ID: 11, Name: "Firo", Ticker: "FIRO"},
	{ID: 12, Name: "Monero", Ticker: "XMR"},
	{ID: 13, Name: "Flux", Ticker: "FLUX"},
	{ID: 14, Name: "Komodo", Ticker: "KMD"},
	{ID: 15, Name: "LBRY Credits", Ticker: "LBC"},
	{ID: 16, Name: "Peercoin", Ticker: "PPC"},
	{ID: 18, Name: "Verge", Ticker: "XVG"},
	{ID: 19, Name: "Zcash", Ticker: "ZEC"},
	{ID: 20, Name: "Horizen", Ticker: "ZEN"},
	{ID: 21, Name: "Ethereum Classic", Ticker: "ETC"},
	{ID: 22, Name: "Raptoreum", Ticker: "RTM"},
	{ID: 23, Name: "Tether", Ticker: "USDT", Networks: []string{"ERC20", "TRC20", "BEP20"}},
	{ID: 24, Name: "Shiba Inu", Ticker: "SHIB"},
	{ID: 25, Name: "Polygon", Ticker: "MATIC"},
	{ID: 26, Name: "Tron", Ticker: "TRX"},
	{ID: 27, Name: "Binance Coin", Ticker: "BNB"},
	{ID: 28, Name: "Cake", Ticker: "CAKE"},
}

var configsMu sync.RWMutex
var configs = DefaultConfigs

// SetConfigs replaces the payout list.
func SetConfigs(c []Config) {
	configsMu.Lock()
	defer configsMu.Unlock()
	configs = c
}

// ByTicker returns the payout currency with this ticker, or false when it is
// not on the list.
func ByTicker(ticker string) (Payout, bool) {
	configsMu.RLock()
	defer configsMu.RUnlock()
	for _, c := range configs {
		if c.Ticker == ticker {
			return NewConfiguredPayout(c), true
		}
	}
	return nil, false
}

// DOGE is the payout to the built-in Dogecoin wallet. The payout list always
// has it.
func DOGE() Payout {
	configsMu.RLock()
	defer configsMu.RUnlock()
	for _, c := range configs {
		if c.ID == DOGEID {
			return NewConfiguredPayout(c)
		}
	}
	return NewConfiguredPayout(Config{ID: DOGEID, Name: "Verthash OCM Dogecoin Wallet", Ticker: "DOGE"})
}
