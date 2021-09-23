package pools

import (
	"fmt"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/util"
	"github.com/vertiond/verthash-one-click-miner/payouts"
)

var _ Pool = &Acidpool{}

type Acidpool struct {
	LastFetchedPayout time.Time
	LastPayout        uint64
}

func NewHAcidpool() *Acidpool {
	return &Acidpool{}
}

func (p *Acidpool) GetPayouts(testnet bool) []payouts.Payout {
	if testnet {
		return []payouts.Payout{
			payouts.NewVTCPayout(),
		}
	}
	return []payouts.Payout{
		payouts.NewDOGEPayout(),
		payouts.NewVTCPayout(),
		payouts.NewBTCPayout(),
		payouts.NewBCHPayout(),
		payouts.NewDASHPayout(),
		payouts.NewDGBPayout(),
		payouts.NewETHPayout(),
		payouts.NewFIROPayout(),
		payouts.NewGRSPayout(),
		payouts.NewLTCPayout(),
		payouts.NewXMRPayout(),
		payouts.NewRVNPayout(),
	}
}

func (p *Acidpool) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("http://acidpool.xyz:4000/api/pools/vtc1/miners/%s", addr), &jsonPayload)
	if err != nil {
		return 0
	}
	vtc, ok := jsonPayload["pendingBalance"].(float64)
	if !ok {
		return 0
	}
	vtc *= 100000000
	return uint64(vtc)
}

func (p *Acidpool) GetStratumUrl() string {
	return "stratum+tcp://vtc.acidpool.co.uk:3052"
}

func (p *Acidpool) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *Acidpool) GetID() int {
	return 8
}

func (p *Acidpool) GetName() string {
	return "Acidpool.co.uk"
}

func (p *Acidpool) GetFee() float64 {
	return 0.69
}

func (p *Acidpool) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("http://acidpool.co.uk/?#vtc1/dashboard?address=%s", addr))
}
