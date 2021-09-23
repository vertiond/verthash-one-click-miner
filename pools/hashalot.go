package pools

import (
	"fmt"
	"time"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &Hashalot{}

type Hashalot struct {
	LastFetchedPayout time.Time
	LastPayout        uint64
}

func NewHashalot() *Hashalot {
	return &Hashalot{}
}

func (p *Hashalot) GetPayouts(testnet bool) []payouts.Payout {
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

func (p *Hashalot) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("http://api.hashalot.net/pools/vtc/miners/%s", addr), &jsonPayload)
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

func (p *Hashalot) GetStratumUrl() string {
	return "stratum+tcp://vertcoin.hashalot.net:3950"
}

func (p *Hashalot) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *Hashalot) GetID() int {
	return 3
}

func (p *Hashalot) GetName() string {
	return "Hashalot.net"
}

func (p *Hashalot) GetFee() float64 {
	return 2.0
}

func (p *Hashalot) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("https://hashalot.net/vtc/miners/%s", addr))
}
