package pools

import (
	"fmt"

	"github.com/vertiond/verthash-one-click-miner/networks"
	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &P2Proxy{}

type P2Proxy struct{}

func NewP2Proxy() *P2Proxy {
	return &P2Proxy{}
}

func (p *P2Proxy) GetPayouts(testnet bool) []payouts.Payout {
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

func (p *P2Proxy) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("%sapi/balance?address=%s", networks.Active.P2ProxyURL, addr), &jsonPayload)
	if err != nil {
		return 0
	}
	vtc, ok := jsonPayload[addr].(float64)
	if !ok {
		return 0
	}
	vtc *= 100000000
	return uint64(vtc)
}

func (p *P2Proxy) GetStratumUrl() string {
	return networks.Active.P2ProxyStratum
}

func (p *P2Proxy) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *P2Proxy) GetID() int {
	return 1
}

func (p *P2Proxy) GetName() string {
	return "P2Proxy"
}

func (p *P2Proxy) GetFee() float64 {
	return 1.00
}

func (p *P2Proxy) OpenBrowserPayoutInfo(addr string) {}
