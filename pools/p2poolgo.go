package pools

import (
	"sync"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/p2poolgo"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

var _ Pool = &P2PoolGo{}

// statusURL is the base URL of the node's API; a var so tests can redirect it.
var statusURL = p2poolgo.LocalURL()

type P2PoolGo struct {
	Address string

	mu                sync.Mutex
	lastFetchedPayout time.Time
	lastPayout        uint64
}

func NewP2PoolGo(addr string) *P2PoolGo {
	return &P2PoolGo{Address: addr}
}

func (p *P2PoolGo) GetPendingPayout() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.lastFetchedPayout) > time.Minute*2 {
		p.lastFetchedPayout = time.Now()
		st, err := p2poolgo.GetStatus(statusURL, p.Address)
		if err != nil || st.ExpectedPayoutSats < 0 {
			if err != nil {
				logging.Warnf("Unable to fetch P2Pool-Go payout: %s", err.Error())
			}
			p.lastPayout = 0
		} else {
			p.lastPayout = uint64(st.ExpectedPayoutSats)
		}
	}
	return p.lastPayout
}

func (p *P2PoolGo) GetStratumUrl() string {
	return "stratum+tcp://127.0.0.1:9172"
}

func (p *P2PoolGo) GetUsername() string {
	return p.Address
}

func (p *P2PoolGo) GetPassword() string {
	return "x"
}

func (p *P2PoolGo) GetID() int {
	return 10
}

func (p *P2PoolGo) GetName() string {
	return "P2Pool-Go"
}

func (p *P2PoolGo) GetFee() float64 {
	return 0
}

func (p *P2PoolGo) OpenBrowserPayoutInfo(addr string) {
	util.OpenBrowser(p2poolgo.LocalURL() + "/")
}
