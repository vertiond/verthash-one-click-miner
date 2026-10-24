package backend

import (
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

// p2pool-go drops shares more than 2 hours ahead of a peer's clock and, from
// its V3 rules, shares older than the median of their last 11 ancestors, which
// is about a minute or two back. A clock off by more than this loses shares.
const (
	clockSkewLimit  = time.Minute
	clockCheckEvery = 10 * time.Minute
)

// Injectable for tests.
var (
	clockReferenceURL = "https://raw.githubusercontent.com/"
	clockSkew         = util.ClockSkew
	clockNow          = time.Now
)

// ClockNotice tells the frontend that this computer's clock is off.
type ClockNotice struct {
	Show    bool `json:"show"`
	Minutes int  `json:"minutes"` // whole minutes off, at least 1 when shown
	Ahead   bool `json:"ahead"`   // the clock is fast rather than slow
}

// GetClockNotice compares the local clock with an HTTPS server's at most every
// 10 minutes and reports when it is off by more than a minute. Without a
// reference time it shows nothing. Bound to the frontend.
func (m *Backend) GetClockNotice() ClockNotice {
	m.clockMu.Lock()
	defer m.clockMu.Unlock()
	now := clockNow()
	if !m.clockChecked.IsZero() && now.Sub(m.clockChecked) < clockCheckEvery {
		return m.clockNotice
	}
	m.clockChecked = now

	skew, err := clockSkew(clockReferenceURL)
	if err != nil {
		logging.Warnf("Unable to check the system clock: %v", err)
		m.clockNotice = ClockNotice{}
		return m.clockNotice
	}
	off := skew
	if off < 0 {
		off = -off
	}
	if off <= clockSkewLimit {
		m.clockNotice = ClockNotice{}
		return m.clockNotice
	}
	minutes := int(off / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	logging.Warnf("System clock is off by %v", skew.Round(time.Second))
	m.clockNotice = ClockNotice{Show: true, Minutes: minutes, Ahead: skew > 0}
	return m.clockNotice
}
