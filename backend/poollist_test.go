package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/pools"
)

func servePoolList(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/pools.json"
}

func poolIDs(configs []pools.Config) []int {
	ids := []int{}
	for _, c := range configs {
		ids = append(ids, c.ID)
	}
	return ids
}

const twoPools = `{
	"payouts": [
		{"id": 4, "name": "Dogecoin Wallet", "ticker": "DOGE"},
		{"id": 30, "name": "Newcoin", "ticker": "NEW", "networks": ["BEP20"]}
	],
	"pools": [
		{"id": 8, "name": "Example", "stratumURL": "stratum+tcp://pool.example:3333", "password": "c={ticker}", "fee": 1.5,
		 "payouts": ["NEW", "DOGE", "GONE"],
		 "payoutURL": "https://pool.example/api/{address}", "payoutFields": ["balance"],
		 "dashboardURL": "https://pool.example/miner/{address}"},
		{"id": 9, "name": "Doge Only", "stratumURL": "stratum+tcp://doge.example:3333", "password": "x", "fee": 0,
		 "payouts": ["DOGE"], "dogeOnly": true, "activationURL": "https://doge.example/"}
	]
}`

func TestLoadPoolManifestUsesFetchedList(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "sub", "pools.json")

	got := loadPoolManifest(servePoolList(t, twoPools), cachePath, fetchPoolList)
	if ids := poolIDs(got.Pools); !reflect.DeepEqual(ids, []int{8, 9}) {
		t.Fatalf("got pools %v, want [8 9]", ids)
	}
	if p := got.Pools[0]; p.Name != "Example" || *p.Fee != 1.5 || !reflect.DeepEqual(p.Payouts, []string{"NEW", "DOGE"}) {
		t.Fatalf("pool 8 = %+v", p)
	}
	if !got.Pools[1].DogeOnly || got.Pools[1].ActivationURL != "https://doge.example/" {
		t.Fatalf("pool 9 = %+v", got.Pools[1])
	}
	if len(got.Payouts) != 2 || got.Payouts[1].Ticker != "NEW" || !reflect.DeepEqual(got.Payouts[1].Networks, []string{"BEP20"}) {
		t.Fatalf("payouts = %+v", got.Payouts)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("the fetched list was not cached: %v", err)
	}
	cached := poolManifest{}
	if err := json.Unmarshal(data, &cached); err != nil || !reflect.DeepEqual(cached, got) {
		t.Fatalf("cached list = %+v (%v), want %+v", cached, err, got)
	}
	if _, err := os.Stat(cachePath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("the temporary cache file was left behind")
	}
}

func TestLoadPoolManifestUsesCacheWhenOffline(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "pools.json")
	loadPoolManifest(servePoolList(t, twoPools), cachePath, fetchPoolList)

	got := loadPoolManifest(offlineURL(), cachePath, fetchPoolList)
	if ids := poolIDs(got.Pools); !reflect.DeepEqual(ids, []int{8, 9}) {
		t.Fatalf("got pools %v, want [8 9]", ids)
	}
}

func TestLoadPoolManifestIgnoresErrorPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(twoPools))
	}))
	t.Cleanup(srv.Close)

	got := loadPoolManifest(srv.URL+"/pools.json", filepath.Join(t.TempDir(), "pools.json"), fetchPoolList)
	if !reflect.DeepEqual(got.Pools, pools.DefaultConfigs) {
		t.Fatalf("got pools %v, want the built-in list", poolIDs(got.Pools))
	}
}

func TestLoadPoolManifestFallsBackToBuiltIn(t *testing.T) {
	for name, body := range map[string]string{
		"offline":  "",
		"not json": "nope",
		"no pools": `{"payouts": [{"id": 4, "name": "Doge", "ticker": "DOGE"}], "pools": []}`,
		"no DOGE payout": `{"payouts": [{"id": 1, "name": "Vertcoin", "ticker": "VTC"}],
			"pools": [{"id": 8, "name": "A", "stratumURL": "stratum+tcp://a.example:1", "password": "x", "fee": 1, "payouts": ["VTC"]}]}`,
		"DOGE with another ID": `{"payouts": [{"id": 5, "name": "Doge", "ticker": "DOGE"}],
			"pools": [{"id": 8, "name": "A", "stratumURL": "stratum+tcp://a.example:1", "password": "x", "fee": 1, "payouts": ["DOGE"]}]}`,
	} {
		listURL := offlineURL()
		if body != "" {
			listURL = servePoolList(t, body)
		}
		got := loadPoolManifest(listURL, filepath.Join(t.TempDir(), "pools.json"), fetchPoolList)
		if !reflect.DeepEqual(got.Pools, pools.DefaultConfigs) || !reflect.DeepEqual(got.Payouts, payouts.DefaultConfigs) {
			t.Errorf("%s: got pools %v, want the built-in list", name, poolIDs(got.Pools))
		}
	}
}

func TestValidPayoutConfigs(t *testing.T) {
	got := validPayoutConfigs([]payouts.Config{
		{ID: 4, Name: "Doge", Ticker: "DOGE"},
		{ID: 0, Name: "No ID", Ticker: "NOID"},
		{ID: 5, Name: "", Ticker: "NONAME"},
		{ID: 6, Name: "Lower", Ticker: "low"},
		{ID: 4, Name: "Same ID", Ticker: "SAME"},
		{ID: 7, Name: "Same ticker", Ticker: "DOGE"},
		{ID: 8, Name: "Bad network", Ticker: "NET", Networks: []string{"ERC-20"}},
		{ID: 9, Name: "Good", Ticker: "GOOD", Networks: []string{"TRC20"}},
	})
	tickers := []string{}
	for _, c := range got {
		tickers = append(tickers, c.Ticker)
	}
	if !reflect.DeepEqual(tickers, []string{"DOGE", "GOOD"}) {
		t.Fatalf("kept %v, want [DOGE GOOD]", tickers)
	}
}

func TestValidPoolConfigs(t *testing.T) {
	ok := func(id int) pools.Config {
		f := 1.0
		return pools.Config{ID: id, Name: "Pool", StratumURL: "stratum+tcp://pool.example:3333", Password: "x", Fee: &f, Payouts: []string{"DOGE"}}
	}
	with := func(id int, change func(*pools.Config)) pools.Config {
		c := ok(id)
		change(&c)
		return c
	}
	badFee := 100.0
	got := validPoolConfigs([]pools.Config{
		ok(10),
		with(2, func(c *pools.Config) {}),
		with(11, func(c *pools.Config) { c.Name = " " }),
		with(12, func(c *pools.Config) { c.StratumURL = "stratum+tcp://pool.example" }),
		with(13, func(c *pools.Config) { c.StratumURL = "https://pool.example:3333" }),
		with(14, func(c *pools.Config) { c.Password = "" }),
		with(15, func(c *pools.Config) { c.Fee = nil }),
		with(16, func(c *pools.Config) { c.Fee = &badFee }),
		with(17, func(c *pools.Config) { c.Payouts = []string{"VTC"} }),
		with(18, func(c *pools.Config) { c.PayoutURL = "http://pool.example/{address}"; c.PayoutFields = []string{"a"} }),
		with(19, func(c *pools.Config) { c.PayoutURL = "https://pool.example/{address}" }),
		with(20, func(c *pools.Config) {
			c.PayoutURL = "https://pool.example/{address}"
			c.PayoutFields = []string{"a..b"}
		}),
		with(21, func(c *pools.Config) { c.DashboardURL = "https://user:pw@pool.example/" }),
		with(22, func(c *pools.Config) { c.ActivationURL = "javascript:alert(1)" }),
		with(23, func(c *pools.Config) { c.ShortageURL = "http://pool.example/api/currencies" }),
		ok(10),
		with(24, func(c *pools.Config) {
			c.Payouts = []string{"VTC", "DOGE"}
			c.PayoutURL = "https://pool.example/api?a={address}"
			c.PayoutFields = []string{"data.unpaid"}
			c.ShortageURL = "https://pool.example/api/currencies"
		}),
	}, map[string]bool{"DOGE": true})
	if ids := poolIDs(got); !reflect.DeepEqual(ids, []int{10, 24}) {
		t.Fatalf("kept %v, want [10 24]", ids)
	}
	if !reflect.DeepEqual(got[1].Payouts, []string{"DOGE"}) {
		t.Fatalf("pool 24 payouts = %v, want the unknown VTC dropped", got[1].Payouts)
	}
}

// pools.json is fetched at startup and the built-in lists are the offline
// fallback, so they should say the same thing.
func TestPublishedPoolListMatchesBuiltIn(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "pools.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := poolManifest{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	valid, ok := validPoolManifest(m)
	if !ok || !reflect.DeepEqual(valid, m) {
		t.Fatalf("pools.json has invalid entries")
	}
	if !reflect.DeepEqual(m.Pools, pools.DefaultConfigs) {
		t.Errorf("pools.json pools differ from pools.DefaultConfigs")
	}
	if !reflect.DeepEqual(m.Payouts, payouts.DefaultConfigs) {
		t.Errorf("pools.json payouts differ from payouts.DefaultConfigs")
	}
}
