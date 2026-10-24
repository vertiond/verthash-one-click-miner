package pools

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestP2PoolGoIdentity(t *testing.T) {
	p := NewP2PoolGo("vtc1qexample")
	if p.GetID() != 10 {
		t.Errorf("id = %d", p.GetID())
	}
	if p.GetName() != "P2Pool-Go" {
		t.Errorf("name = %q", p.GetName())
	}
	if p.GetFee() != 0 {
		t.Errorf("fee = %v", p.GetFee())
	}
	if p.GetStratumUrl() != "stratum+tcp://127.0.0.1:9172" {
		t.Errorf("stratum = %q", p.GetStratumUrl())
	}
	if p.GetUsername() != "vtc1qexample" {
		t.Errorf("username = %q", p.GetUsername())
	}
	if p.GetPassword() != "x" {
		t.Errorf("password = %q", p.GetPassword())
	}
	if GetPool("abc").GetID() != 10 {
		t.Error("GetPool should return the p2pool-go pool")
	}
}

func TestP2PoolGoPendingPayout(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(`{"ready":true,"state":"ready","version":"1.0","expected_payout_sats":12345}`))
	}))
	defer srv.Close()
	old := statusURL
	statusURL = srv.URL
	defer func() { statusURL = old }()

	p := NewP2PoolGo("vtc1qexample")
	if got := p.GetPendingPayout(); got != 12345 {
		t.Fatalf("payout = %d", got)
	}
	if got := p.GetPendingPayout(); got != 12345 {
		t.Fatalf("cached payout = %d", got)
	}
	if h := atomic.LoadInt32(&hits); h != 1 {
		t.Errorf("server hit %d times, want 1", h)
	}
}
