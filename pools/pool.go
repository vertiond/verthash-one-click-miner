package pools

import (
	"github.com/vertiond/verthash-one-click-miner/payouts"
)

type Pool interface {
	GetPayouts(testnet bool) []payouts.Payout
	GetPendingPayout(addr string) uint64
	GetStratumUrl() string
	GetPassword(payoutTicker string) string
	GetName() string
	GetID() int
	GetFee() float64
	OpenBrowserPayoutInfo(addr string)
}

func GetPools(testnet bool) []Pool {
	if testnet {
		return []Pool{
			NewP2Proxy(),
		}
	}
	return []Pool{
		NewZergpool(),
		Newzpool(),
		NewHashCryptos(),
		//NewHashalot(),
		//NewSuprnova(),
		//NewP2Pool(),
		//NewBBQDroid(addr),
		//NewAcidpool(addr),
	}
}

func GetPool(poolID int, testnet bool) Pool {
	pools := GetPools(testnet)
	for _, p := range pools {
		if p.GetID() == poolID {
			return p
		}
	}
	return pools[0]
}

func GetPayout(pool Pool, payoutID int, testnet bool) payouts.Payout {
	payouts := pool.GetPayouts(testnet)
	for _, p := range payouts {
		if p.GetID() == payoutID {
			return p
		}
	}
	return payouts[0]
}
