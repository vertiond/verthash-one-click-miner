package pools

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &ConfiguredPool{}
var _ ShortageReporter = &ConfiguredPool{}
var _ DogeOnlyReporter = &ConfiguredPool{}

const shortageCacheTTL = 5 * time.Minute

// ConfiguredPool is a pool described entirely by a pools.json entry.
type ConfiguredPool struct {
	Config Config

	shortageLock    sync.Mutex
	shortageFetched time.Time
	shortages       map[string]bool
}

func NewConfiguredPool(c Config) *ConfiguredPool {
	return &ConfiguredPool{Config: c}
}

func withAddress(template, addr string) string {
	return strings.ReplaceAll(template, "{address}", url.QueryEscape(addr))
}

// GetPayouts returns the pool's payout currencies that are on the payout list,
// in the pool's order.
func (p *ConfiguredPool) GetPayouts(testnet bool) []payouts.Payout {
	if testnet {
		return []payouts.Payout{
			payouts.VTC(),
		}
	}
	list := make([]payouts.Payout, 0, len(p.Config.Payouts))
	for _, ticker := range p.Config.Payouts {
		if payout, ok := payouts.ByTicker(ticker); ok {
			list = append(list, payout)
		}
	}
	if len(list) == 0 {
		list = append(list, payouts.DOGE())
	}
	return list
}

func (p *ConfiguredPool) GetPendingPayout(addr string) uint64 {
	if p.Config.PayoutURL == "" || len(p.Config.PayoutFields) == 0 {
		return 0
	}
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(withAddress(p.Config.PayoutURL, addr), &jsonPayload)
	if err != nil {
		return 0
	}
	unpaid := 0.0
	for _, field := range p.Config.PayoutFields {
		v, ok := lookupNumber(jsonPayload, field)
		if !ok {
			return 0
		}
		unpaid += v
	}
	unpaid *= 100000000
	return uint64(unpaid)
}

// lookupNumber follows a dot-separated path of object keys and returns the
// number found there, which may also be written as a string.
func lookupNumber(payload map[string]interface{}, path string) (float64, bool) {
	var el interface{} = payload
	for _, key := range strings.Split(path, ".") {
		obj, ok := el.(map[string]interface{})
		if !ok {
			return 0, false
		}
		el = obj[key]
	}
	switch v := el.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

// PayoutShortage reports whether the pool says it is short of the given
// currency, meaning payouts in it may be delayed. Pools without a ShortageURL,
// guaranteed payout currencies and any failure to find out report false.
func (p *ConfiguredPool) PayoutShortage(ticker string) bool {
	if p.Config.ShortageURL == "" {
		return false
	}
	for _, guaranteed := range p.Config.GuaranteedPayouts {
		if guaranteed == ticker {
			return false
		}
	}

	p.shortageLock.Lock()
	defer p.shortageLock.Unlock()

	if p.shortages == nil || time.Since(p.shortageFetched) > shortageCacheTTL {
		currencies := map[string]struct {
			Error string `json:"error"`
		}{}
		// Retry on the next call rather than hammering the API while it is down
		p.shortageFetched = time.Now()
		if err := util.GetJson(p.Config.ShortageURL, &currencies); err != nil {
			p.shortages = map[string]bool{}
			return false
		}
		p.shortages = make(map[string]bool, len(currencies))
		for symbol, currency := range currencies {
			p.shortages[symbol] = currency.Error != ""
		}
	}
	return p.shortages[ticker]
}

func (p *ConfiguredPool) GetStratumUrl() string {
	return p.Config.StratumURL
}

func (p *ConfiguredPool) GetPassword(payoutTicker string, network string) string {
	return strings.NewReplacer("{ticker}", payoutTicker, "{network}", network).Replace(p.Config.Password)
}

func (p *ConfiguredPool) GetID() int {
	return p.Config.ID
}

func (p *ConfiguredPool) GetName() string {
	return p.Config.Name
}

func (p *ConfiguredPool) GetFee() float64 {
	if p.Config.Fee == nil {
		return 0
	}
	return *p.Config.Fee
}

func (p *ConfiguredPool) OpenBrowserPayoutInfo(addr string) {
	if p.Config.DashboardURL == "" {
		return
	}
	util.OpenBrowser(withAddress(p.Config.DashboardURL, addr))
}

func (p *ConfiguredPool) DogeOnly() bool {
	return p.Config.DogeOnly
}

func (p *ConfiguredPool) ActivationURL() string {
	return p.Config.ActivationURL
}
