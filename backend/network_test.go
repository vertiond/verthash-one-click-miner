package backend

import (
	"testing"

	"github.com/tidwall/buntdb"
	"github.com/vertiond/verthash-one-click-miner/payouts"
)

func TestNetworkIgnoredForPayoutWithoutNetworks(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := &Backend{settings: db, payout: payouts.VTC()}

	m.SetNetwork("TRC20")
	if got := m.getStringSetting("network"); got != "" {
		t.Errorf("network saved for VTC = %q, want nothing saved", got)
	}
	if m.network != "" {
		t.Errorf("network kept for VTC = %q, want \"\"", m.network)
	}
}

func TestNetworkSavedForPayoutWithNetworks(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	usdt := payouts.NewConfiguredPayout(payouts.Config{ID: 23, Name: "Tether", Ticker: "USDT", Networks: []string{"ERC20", "TRC20", "BEP20"}})
	m := &Backend{settings: db, payout: usdt}

	m.SetNetwork("TRC20")
	if got := m.getStringSetting("network"); got != "TRC20" {
		t.Errorf("network saved for USDT = %q, want TRC20", got)
	}

	// After a restart the saved network is loaded back
	restarted := &Backend{settings: db, payout: usdt}
	restarted.ResetNetwork()
	if !restarted.ChainNetworkApplicableToPayout() || restarted.network != "TRC20" {
		t.Errorf("network after restart = %q, want TRC20", restarted.network)
	}
}
