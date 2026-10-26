package pools

import (
	"sync"

	"github.com/vertiond/verthash-one-click-miner/payouts"
)

type Pool interface {
	GetPayouts() []payouts.Payout
	GetPendingPayout(addr string) uint64
	GetStratumUrl() string
	GetPassword(payoutTicker string, network string) string
	GetName() string
	GetID() int
	GetFee() float64
	OpenBrowserPayoutInfo(addr string)
}

// ShortageReporter is implemented by pools that can tell when they are unable
// to pay out in a currency.
type ShortageReporter interface {
	PayoutShortage(ticker string) bool
}

// DogeOnlyReporter is implemented by pools that can only pay out to the
// built-in Dogecoin wallet.
type DogeOnlyReporter interface {
	DogeOnly() bool
	ActivationURL() string
}

// IsDogeOnly reports whether the pool only pays out to the built-in Dogecoin
// wallet.
func IsDogeOnly(p Pool) bool {
	r, ok := p.(DogeOnlyReporter)
	return ok && r.DogeOnly()
}

// Config describes one mainnet pool in pools.json. In Password, {ticker} is
// replaced by the payout ticker and {network} by "-" and the chain network
// when one applies. In PayoutURL and DashboardURL, {address} is replaced by
// the mining address. The pending payout is the sum of the PayoutFields, each
// a dot-separated path into the PayoutURL's JSON response. Payouts lists the
// tickers of the payout currencies the pool offers. DogeOnly pools always pay
// the built-in Dogecoin wallet, which the user may first have to activate at
// ActivationURL. ShortageURL is a YIIMP-style /api/currencies endpoint whose
// "error" field marks a currency the pool is short of, except for the
// GuaranteedPayouts it tops up daily.
type Config struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	StratumURL        string   `json:"stratumURL"`
	Password          string   `json:"password"`
	Fee               *float64 `json:"fee"`
	Payouts           []string `json:"payouts"`
	PayoutURL         string   `json:"payoutURL,omitempty"`
	PayoutFields      []string `json:"payoutFields,omitempty"`
	DashboardURL      string   `json:"dashboardURL,omitempty"`
	DogeOnly          bool     `json:"dogeOnly,omitempty"`
	ActivationURL     string   `json:"activationURL,omitempty"`
	ShortageURL       string   `json:"shortageURL,omitempty"`
	GuaranteedPayouts []string `json:"guaranteedPayouts,omitempty"`
}

func fee(f float64) *float64 { return &f }

// DefaultConfigs is the mainnet pool list used when neither pools.json nor its
// cached copy is available.
var DefaultConfigs = []Config{
	{
		ID:         7,
		Name:       "Zpool.ca",
		StratumURL: "stratum+tcp://verthash.mine.zpool.ca:6144",
		Password:   "c={ticker},zap=VTC",
		Fee:        fee(0.50),
		Payouts: []string{
			"DOGE", "VTC", "BTC", "DASH", "DGB", "FLUX", "GRS", "ZEN",
			"KMD", "LTC", "PPC", "RVN", "XVG", "ZEC",
		},
		PayoutURL:         "https://zpool.ca/api/wallet?address={address}",
		PayoutFields:      []string{"unpaid"},
		DashboardURL:      "https://zpool.ca/wallet/{address}",
		ShortageURL:       "https://zpool.ca/api/currencies",
		GuaranteedPayouts: []string{"BTC", "LTC", "DASH", "DGB", "FLUX", "RVN"},
	},
	{
		ID:         6,
		Name:       "HashCryptos.com",
		StratumURL: "stratum+tcp://stratum3.hashcryptos.com:9991",
		Password:   "x",
		Fee:        fee(0.00),
		Payouts: []string{
			"DOGE", "VTC", "BTC", "BCH", "DASH", "DGB", "FIRO", "GRS",
			"LTC", "XMR", "RVN",
		},
		PayoutURL:     "https://www.hashcryptos.com/api/walletEx/?address={address}",
		PayoutFields:  []string{"unpaid"},
		DashboardURL:  "https://hashcryptos.com/?address={address}",
		DogeOnly:      true,
		ActivationURL: "https://hashcryptos.com",
	},
}

var configsMu sync.RWMutex
var mainnetConfigs = DefaultConfigs

// The configured pools keep their shortage cache between calls to GetPools.
var configuredPools = map[int]*ConfiguredPool{}

// SetConfigs replaces the mainnet pool list.
func SetConfigs(configs []Config) {
	configsMu.Lock()
	defer configsMu.Unlock()
	mainnetConfigs = configs
	configuredPools = map[int]*ConfiguredPool{}
}

func GetPools() []Pool {
	configsMu.Lock()
	defer configsMu.Unlock()
	pools := make([]Pool, 0, len(mainnetConfigs))
	for _, c := range mainnetConfigs {
		p, ok := configuredPools[c.ID]
		if !ok {
			p = NewConfiguredPool(c)
			configuredPools[c.ID] = p
		}
		pools = append(pools, p)
	}
	//pools = append(pools, NewP2Pool())
	return pools
}

func GetPool(poolID int) Pool {
	pools := GetPools()
	for _, p := range pools {
		if p.GetID() == poolID {
			return p
		}
	}
	return pools[0]
}

// HasPool reports whether the pool with this ID is on the current list.
func HasPool(poolID int) bool {
	for _, p := range GetPools() {
		if p.GetID() == poolID {
			return true
		}
	}
	return false
}

func GetPayout(pool Pool, payoutID int) payouts.Payout {
	payouts := pool.GetPayouts()
	for _, p := range payouts {
		if p.GetID() == payoutID {
			return p
		}
	}
	return payouts[0]
}
