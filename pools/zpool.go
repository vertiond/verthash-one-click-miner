package pools

import (
	"fmt"
	"sync"
	"time"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &zpool{}
var _ ShortageReporter = &zpool{}

const shortageCacheTTL = 5 * time.Minute

// zpool tops these payout currencies up daily, so a shortage is only a short delay
var guaranteedPayouts = map[string]bool{
	"BTC": true, "LTC": true, "DASH": true, "DGB": true, "FLUX": true, "RVN": true,
}

type zpool struct {
	LastFetchedPayout time.Time
	LastPayout        uint64

	shortageLock    sync.Mutex
	shortageFetched time.Time
	shortages       map[string]bool
}

func Newzpool() *zpool {
	return &zpool{}
}

func (p *zpool) GetPayouts(testnet bool) []payouts.Payout {
	if testnet {
		return []payouts.Payout{
			payouts.NewVTCPayout(),
		}
	}
	return []payouts.Payout{
		payouts.NewDOGEPayout(),
		payouts.NewVTCPayout(),
		payouts.NewBTCPayout(),
		payouts.NewDASHPayout(),
		payouts.NewDGBPayout(),
		payouts.NewFLUXPayout(),
		payouts.NewGRSPayout(),
		payouts.NewZENPayout(),
		payouts.NewKMDPayout(),
		payouts.NewLTCPayout(),
		payouts.NewPPCPayout(),
		payouts.NewRVNPayout(),
		payouts.NewXVGPayout(),
		payouts.NewZECPayout(),
	}
}

func (p *zpool) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("https://zpool.ca/api/wallet?address=%s", addr), &jsonPayload)
	if err != nil {
		return 0
	}
	vtc, ok := jsonPayload["unpaid"].(float64)
	if !ok {
		return 0
	}
	vtc *= 100000000
	return uint64(vtc)
}

// PayoutShortage reports whether zpool says it is short of the given currency,
// meaning payouts in it may be delayed. Guaranteed payout currencies and any
// failure to find out report false.
func (p *zpool) PayoutShortage(ticker string) bool {
	if guaranteedPayouts[ticker] {
		return false
	}

	p.shortageLock.Lock()
	defer p.shortageLock.Unlock()

	if p.shortages == nil || time.Since(p.shortageFetched) > shortageCacheTTL {
		currencies := map[string]struct {
			Error string `json:"error"`
		}{}
		// Retry on the next call rather than hammering the API while it is down
		p.shortageFetched = time.Now()
		if err := util.GetJson("https://zpool.ca/api/currencies", &currencies); err != nil {
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

func (p *zpool) GetStratumUrl() string {
	return "stratum+tcp://verthash.mine.zpool.ca:6144"
}

func (p *zpool) GetPassword(payoutTicker string, network string) string {
	return fmt.Sprintf("c=%s,zap=VTC", payoutTicker)
}

func (p *zpool) GetID() int {
	return 7
}

func (p *zpool) GetName() string {
	return "Zpool.ca"
}

func (p *zpool) GetFee() float64 {
	return 0.50
}

func (p *zpool) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("https://zpool.ca/wallet/%s", addr))
}
