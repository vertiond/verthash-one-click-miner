package pools

import (
	"fmt"
	"time"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &Suprnova{}

type Suprnova struct {
	LastFetchedPayout time.Time
	LastPayout        uint64
}

func NewSuprnova() *Suprnova {
	return &Suprnova{}
}

func (p *Suprnova) GetPayouts(testnet bool) []payouts.Payout {
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

func (p *Suprnova) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("https://vtc.suprnova.cc/index.php?page=api&action=getuserbalance&api_key=%s", addr), &jsonPayload)
	if err != nil {
		return 0
	}
	el, ok := jsonPayload["getuserbalance"].(map[string]interface{})
	if !ok {
		return 0
	}
	el, ok = el["data"].(map[string]interface{})
	if !ok {
		return 0
	}

	confirmed, ok := el["confirmed"].(float64)
	if !ok {
		return 0
	}

	unconfirmed, ok := el["unconfirmed"].(float64)
	if !ok {
		return 0
	}

	vtc := confirmed + unconfirmed
	vtc *= 100000000
	return uint64(vtc)
}

func (p *Suprnova) GetStratumUrl() string {
	return "stratum+tcp://vtc.suprnova.cc:1776"
}

func (p *Suprnova) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *Suprnova) GetID() int {
	return 4
}

func (p *Suprnova) GetName() string {
	return "Suprnova.cc"
}

func (p *Suprnova) GetFee() float64 {
	return 1.00
}

func (p *Suprnova) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("https://vtc.suprnova.cc/index.php?page=anondashboard&user=%s", addr))
}
