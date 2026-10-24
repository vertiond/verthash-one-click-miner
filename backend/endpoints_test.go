package backend

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vertiond/verthash-one-click-miner/util"
)

func serveEndpoints(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const goodEndpoints = `{
	"blockbookUrls": ["https://bb1.example/", "https://bb2.example"],
	"coinpaprikaUrl": "https://prices.example/v1/",
	"coinpaprikaIds": {"DOGE": "doge-dogecoin-2"}
}`

var goodEndpointList = endpointList{
	BlockbookURLs:  []string{"https://bb1.example/", "https://bb2.example/"},
	CoinpaprikaURL: "https://prices.example/v1",
	CoinpaprikaIDs: map[string]string{"DOGE": "doge-dogecoin-2"},
}

func TestFetchedEndpointsAreUsedAndCached(t *testing.T) {
	srv := serveEndpoints(t, goodEndpoints)
	cachePath := filepath.Join(t.TempDir(), "backends.json")

	got := loadEndpoints(srv.URL, cachePath, util.GetJson)
	if !reflect.DeepEqual(got, goodEndpointList) {
		t.Fatalf("got %+v, want %+v", got, goodEndpointList)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("no cache written: %v", err)
	}
	cached := endpointList{}
	if err := json.Unmarshal(data, &cached); err != nil || !reflect.DeepEqual(cached, goodEndpointList) {
		t.Fatalf("cache holds %s", data)
	}
	if _, err := os.Stat(cachePath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary cache file left behind")
	}
}

func TestCachedEndpointsAreUsedOffline(t *testing.T) {
	srv := serveEndpoints(t, goodEndpoints)
	cachePath := filepath.Join(t.TempDir(), "backends.json")
	loadEndpoints(srv.URL, cachePath, util.GetJson)
	srv.Close()

	if got := loadEndpoints(srv.URL, cachePath, util.GetJson); !reflect.DeepEqual(got, goodEndpointList) {
		t.Fatalf("got %+v, want the cached %+v", got, goodEndpointList)
	}
}

func TestCachedEndpointsAreUsedWhenTheFetchedOnesAreAllInvalid(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "backends.json")
	loadEndpoints(serveEndpoints(t, goodEndpoints).URL, cachePath, util.GetJson)

	for name, body := range map[string]string{
		"not json":    "<html>",
		"empty":       `{}`,
		"all invalid": `{"blockbookUrls": ["http://bb.example/"], "coinpaprikaIds": {"DOGE": ""}}`,
	} {
		if got := loadEndpoints(serveEndpoints(t, body).URL, cachePath, util.GetJson); !reflect.DeepEqual(got, goodEndpointList) {
			t.Errorf("%s: got %+v, want the cached %+v", name, got, goodEndpointList)
		}
	}
}

func TestNoEndpointsWithoutAFetchOrACache(t *testing.T) {
	srv := serveEndpoints(t, goodEndpoints)
	srv.Close()
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "backends.json")
	os.WriteFile(cachePath, []byte("corrupt"), 0644)

	if got := loadEndpoints(srv.URL, cachePath, util.GetJson); !got.empty() {
		t.Fatalf("got %+v, want nothing so the built-in defaults stay", got)
	}
	if got := loadEndpoints(srv.URL, filepath.Join(dir, "missing.json"), util.GetJson); !got.empty() {
		t.Fatalf("got %+v, want nothing so the built-in defaults stay", got)
	}
}

func TestInvalidEndpointsAreDropped(t *testing.T) {
	got := validEndpoints(endpointList{
		BlockbookURLs: []string{
			"https://ok.example/api",
			"http://plain.example/",
			"https://user:pass@creds.example/",
			"https://query.example/?x=1",
			"https://fragment.example/#x",
			"https:///nohost",
			"ftp://ftp.example/",
			"",
		},
		CoinpaprikaURL: "http://prices.example/v1",
		CoinpaprikaIDs: map[string]string{
			"VTC":  "vtc-vertcoin",
			"":     "no-ticker",
			"DOGE": "",
			"BAD":  "../../tickers",
			"UP":   "UPPER-case",
		},
	})
	want := endpointList{
		BlockbookURLs:  []string{"https://ok.example/api/"},
		CoinpaprikaIDs: map[string]string{"VTC": "vtc-vertcoin"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// backends.json in the repository is the file the app fetches, so all of it
// has to pass validation unchanged.
func TestPublishedEndpointsAreValid(t *testing.T) {
	data, err := os.ReadFile("../backends.json")
	if err != nil {
		t.Fatal(err)
	}
	published := endpointList{}
	if err := json.Unmarshal(data, &published); err != nil {
		t.Fatal(err)
	}
	if got := validEndpoints(published); !reflect.DeepEqual(got, published) ||
		len(published.BlockbookURLs) == 0 || published.CoinpaprikaURL == "" || len(published.CoinpaprikaIDs) == 0 {
		t.Fatalf("backends.json has invalid or missing entries: published %+v, valid %+v", published, got)
	}
}
