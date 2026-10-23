package payouts

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func useCoinpaprika(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	old := coinpaprikaURL
	coinpaprikaURL = srv.URL
	t.Cleanup(func() {
		srv.Close()
		coinpaprikaURL = old
	})
}

func TestBitcoinPerUnitCoin(t *testing.T) {
	var path, query string
	useCoinpaprika(t, func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"id":"doge-dogecoin","quotes":{"BTC":{"price":9.7e-7}}}`)
	})

	got := GetBitcoinPerUnitCoin("DOGE")
	if got != 9.7e-7 {
		t.Fatalf("price = %v, want 9.7e-7", got)
	}
	if path != "/tickers/doge-dogecoin" || query != "quotes=BTC" {
		t.Fatalf("unexpected request %s?%s", path, query)
	}
}

func TestBitcoinIsOneBitcoinWithoutALookup(t *testing.T) {
	useCoinpaprika(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("BTC must not need a lookup")
	})
	if got := GetBitcoinPerUnitCoin("BTC"); got != 1.0 {
		t.Fatalf("price = %v, want 1", got)
	}
}

func TestUnknownCoinHasNoPrice(t *testing.T) {
	useCoinpaprika(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unknown coins must not be looked up")
	})
	if got := GetBitcoinPerUnitCoin("GRS"); got != 0 {
		t.Fatalf("price = %v, want 0", got)
	}
}

func TestFailedLookupHasNoPrice(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"not json":      func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "nope") },
		"no BTC quote":  func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"quotes":{}}`) },
		"unknown error": func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"error":"id not found"}`) },
	} {
		useCoinpaprika(t, handler)
		if got := GetBitcoinPerUnitCoin("VTC"); got != 0 {
			t.Errorf("%s: price = %v, want 0", name, got)
		}
	}
}

func TestEveryPayoutCoinHasAPriceSourceOrIsKnownToLackOne(t *testing.T) {
	// GRS is delisted on CoinPaprika. Every other payout must be mapped.
	lacking := map[string]bool{"GRS": true}
	for _, p := range []Payout{
		NewBCHPayout(), NewBNBPayout(), NewBTCPayout(), NewCAKEPayout(), NewDASHPayout(),
		NewDGBPayout(), NewDOGEPayout(), NewETCPayout(), NewFIROPayout(), NewFLUXPayout(),
		NewGRSPayout(), NewKMDPayout(), NewLBCPayout(), NewLTCPayout(), NewMATICPayout(),
		NewPPCPayout(), NewRTMPayout(), NewRVNPayout(), NewSHIBPayout(), NewTRXPayout(),
		NewUSDTPayout(), NewVTCPayout(), NewXMRPayout(), NewXVGPayout(), NewZECPayout(),
		NewZENPayout(),
	} {
		_, mapped := coinpaprikaIDs[p.GetTicker()]
		if !mapped && !lacking[p.GetTicker()] {
			t.Errorf("%s has no CoinPaprika id", p.GetTicker())
		}
	}
}
