package payouts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
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
	if got := GetBitcoinPerUnitCoin("NOPE"); got != 0 {
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
	// Every payout must be mapped. Add a ticker here only if CoinPaprika delists it.
	lacking := map[string]bool{}
	for _, c := range DefaultConfigs {
		p := NewConfiguredPayout(c)
		_, mapped := coinpaprikaIDs[p.GetTicker()]
		if !mapped && !lacking[p.GetTicker()] {
			t.Errorf("%s has no CoinPaprika id", p.GetTicker())
		}
	}
}

func TestSetCoinpaprika(t *testing.T) {
	oldURL, oldIDs := coinpaprikaURL, coinpaprikaIDs
	t.Cleanup(func() { coinpaprikaURL, coinpaprikaIDs = oldURL, oldIDs })

	SetCoinpaprika("", nil)
	if coinpaprikaURL != DefaultCoinpaprikaURL || !reflect.DeepEqual(coinpaprikaIDs, oldIDs) {
		t.Fatalf("empty values replaced the defaults")
	}

	SetCoinpaprika("https://prices.example/v1", map[string]string{"DOGE": "doge-new", "NEW": "new-coin"})
	if coinpaprikaURL != "https://prices.example/v1" {
		t.Fatalf("coinpaprikaURL = %s", coinpaprikaURL)
	}
	if coinpaprikaIDs["DOGE"] != "doge-new" || coinpaprikaIDs["NEW"] != "new-coin" || coinpaprikaIDs["VTC"] != "vtc-vertcoin" {
		t.Fatalf("ids not merged over the defaults: %v", coinpaprikaIDs)
	}
	if oldIDs["DOGE"] != "doge-dogecoin" {
		t.Fatalf("the built-in ids were modified in place")
	}
}

// backends.json is fetched at startup, the built-in ids are the offline
// fallback; keep them in step.
func TestPublishedCoinpaprikaMatchesTheBuiltInOne(t *testing.T) {
	data, err := os.ReadFile("../backends.json")
	if err != nil {
		t.Fatal(err)
	}
	published := struct {
		CoinpaprikaURL string            `json:"coinpaprikaUrl"`
		CoinpaprikaIDs map[string]string `json:"coinpaprikaIds"`
	}{}
	if err := json.Unmarshal(data, &published); err != nil {
		t.Fatal(err)
	}
	if published.CoinpaprikaURL != DefaultCoinpaprikaURL {
		t.Errorf("backends.json has coinpaprikaUrl %s, the default is %s", published.CoinpaprikaURL, DefaultCoinpaprikaURL)
	}
	if !reflect.DeepEqual(published.CoinpaprikaIDs, coinpaprikaIDs) {
		t.Errorf("backends.json has coinpaprikaIds %v, the defaults are %v", published.CoinpaprikaIDs, coinpaprikaIDs)
	}
}
