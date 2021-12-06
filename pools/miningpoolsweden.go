package pools

import (
	"fmt"
	"time"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

var _ Pool = &MiningpoolSweden{}

type MiningpoolSweden struct {
	LastFetchedPayout time.Time
	LastPayout        uint64
}

func NewHMiningpoolSweden() *MiningpoolSweden {
	return &MiningpoolSweden{}
}

func (p *MiningpoolSweden) GetPayouts(testnet bool) []payouts.Payout {
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

func (p *MiningpoolSweden) GetPendingPayout(addr string) uint64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(fmt.Sprintf("https://api.miningpoolsweden.eu/api/pools/vert1/miners/%s", addr), &jsonPayload)
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

func (p *MiningpoolSweden) GetStratumUrl() string {
	return "stratum+tcp://vtc.miningpoolsweden.eu:3052"
}

func (p *MiningpoolSweden) GetPassword(payoutTicker string) string {
	return "x"
}

func (p *MiningpoolSweden) GetID() int {
	return 9
}

func (p *MiningpoolSweden) GetName() string {
	return "MiningpoolSweden.eu"
}

func (p *MiningpoolSweden) GetFee() float64 {
	return 0.6
}

func (p *MiningpoolSweden) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(fmt.Sprintf("https://miningpoolsweden.eu/?#vert1/dashboard?address=%s", addr))
}
