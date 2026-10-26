package pools

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/vertiond/verthash-one-click-miner/payouts"
)

func TestConfiguredPoolPendingPayout(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Write([]byte(`{"data": {"confirmed": 1.25, "unconfirmed": 0.5}, "unpaid": "1.5", "bad": "x"}`))
	}))
	defer srv.Close()

	p := NewConfiguredPool(Config{
		PayoutURL:    srv.URL + "/api?key={address}",
		PayoutFields: []string{"data.confirmed", "data.unconfirmed"},
	})
	if got := p.GetPendingPayout("D8addr"); got != 175000000 {
		t.Fatalf("payout = %d, want 175000000", got)
	}
	if gotPath != "/api?key=D8addr" {
		t.Fatalf("requested %q", gotPath)
	}

	// HashCryptos sends the balance as a string
	p.Config.PayoutFields = []string{"unpaid"}
	if got := p.GetPendingPayout("D8addr"); got != 150000000 {
		t.Fatalf("string payout = %d, want 150000000", got)
	}

	for _, fields := range [][]string{
		{"data.missing"},
		{"data.confirmed.deeper"},
		{"bad"},
	} {
		p.Config.PayoutFields = fields
		if got := p.GetPendingPayout("D8addr"); got != 0 {
			t.Errorf("fields %v: payout = %d, want 0", fields, got)
		}
	}

	p.Config.PayoutURL = ""
	if got := p.GetPendingPayout("D8addr"); got != 0 {
		t.Errorf("no payout URL: payout = %d, want 0", got)
	}
}

func TestConfiguredPoolPassword(t *testing.T) {
	p := NewConfiguredPool(Config{Password: "c={ticker}{network},zap=VTC"})
	if got := p.GetPassword("USDT", "-TRC20"); got != "c=USDT-TRC20,zap=VTC" {
		t.Fatalf("password = %q", got)
	}
	if got := p.GetPassword("DOGE", ""); got != "c=DOGE,zap=VTC" {
		t.Fatalf("password = %q", got)
	}
}

func TestConfiguredPoolPayoutShortage(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Write([]byte(`{"DOGE": {"error": "low balance"}, "LTC": {"error": "low balance"}, "VTC": {"error": ""}}`))
	}))
	defer srv.Close()

	p := NewConfiguredPool(Config{ShortageURL: srv.URL, GuaranteedPayouts: []string{"LTC"}})
	if !p.PayoutShortage("DOGE") {
		t.Errorf("DOGE: want a shortage")
	}
	if p.PayoutShortage("VTC") || p.PayoutShortage("LTC") || p.PayoutShortage("BTC") {
		t.Errorf("VTC, guaranteed LTC and unlisted BTC: want no shortage")
	}
	if n := atomic.LoadInt32(&requests); n != 1 {
		t.Errorf("fetched the currencies %d times, want once", n)
	}

	if NewConfiguredPool(Config{}).PayoutShortage("DOGE") {
		t.Errorf("a pool without a shortage URL reported a shortage")
	}
}

func TestConfiguredPoolPayouts(t *testing.T) {
	t.Cleanup(func() { payouts.SetConfigs(payouts.DefaultConfigs) })
	payouts.SetConfigs([]payouts.Config{
		{ID: 4, Name: "Doge", Ticker: "DOGE"},
		{ID: 23, Name: "Tether", Ticker: "USDT", Networks: []string{"TRC20"}},
	})

	got := NewConfiguredPool(Config{Payouts: []string{"USDT", "GONE", "DOGE"}}).GetPayouts()
	if len(got) != 2 || got[0].GetID() != 23 || got[1].GetTicker() != "DOGE" || got[0].GetNetworks()[0] != "TRC20" {
		t.Fatalf("payouts = %+v", got)
	}
}

func TestDefaultPools(t *testing.T) {
	t.Cleanup(func() { SetConfigs(DefaultConfigs) })
	SetConfigs(DefaultConfigs)

	if !HasPool(7) || !HasPool(6) || HasPool(5) {
		t.Fatalf("expected Zpool (7) and HashCryptos (6) only")
	}
	if IsDogeOnly(GetPool(7)) || !IsDogeOnly(GetPool(6)) {
		t.Fatalf("only HashCryptos should be DOGE only")
	}
	if GetPool(5).GetID() != 7 {
		t.Fatalf("an unknown pool should fall back to the first one")
	}
	if GetPool(7) != GetPool(7) {
		t.Fatalf("GetPools should keep the same pool so its shortage cache lasts")
	}
}
