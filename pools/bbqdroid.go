package pools

import (
	"fmt"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/util"
	"github.com/vertiond/verthash-one-click-miner/payouts"
)

var _ Pool = &BBQDroid{}

type BBQDroid struct {
	LastFetchedPayout time.Time
	LastPayout        uint64
}

func NewBBQDroid() *BBQDroid {
	return &BBQDroid{}
}

func (p *BBQDroid) GetPayouts(testnet bool) []payouts.Payout {
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

func (p *BBQDroid) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("https://miningapi.bbqdroid.org/api/pools/vertcoin/miners/%s", addr), &jsonPayload)
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

func (p *BBQDroid) GetStratumUrl() string {
	return "stratum+tcp://bbqdroid.org:10001"
}

func (p *BBQDroid) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *BBQDroid) GetID() int {
	return 7
}

func (p *BBQDroid) GetName() string {
	return "BBQDroid.org"
}

func (p *BBQDroid) GetFee() float64 {
	return 0.5
}

func (p *BBQDroid) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("https://bbqdroid.org/?#vertcoin/dashboard?address=%s", addr))
}
