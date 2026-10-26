package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/pools"
	"github.com/vertiond/verthash-one-click-miner/util"
)

// poolListURL is published next to miners.json, so pools and payout
// currencies can change without a release.
const poolListURL = "https://raw.githubusercontent.com/vertiond/verthash-one-click-miner/master/pools.json"

// The pool list is fetched before the window opens, so give up quickly and
// fall back to the cached copy.
var poolListClient = &http.Client{Timeout: 5 * time.Second}

func fetchPoolList(listURL string, target interface{}) error {
	r, err := poolListClient.Get(listURL)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", r.StatusCode)
	}
	return json.NewDecoder(r.Body).Decode(target)
}

// poolManifest is the format of pools.json. The order of Pools is the order
// shown in the settings, and the order of a pool's payouts is the order of its
// payout menu.
type poolManifest struct {
	Payouts []payouts.Config `json:"payouts"`
	Pools   []pools.Config   `json:"pools"`
}

// ResolvePools loads the mainnet pool and payout lists from pools.json (or its
// cached copy) and makes them active. Run it before ResetPool.
func (m *Backend) ResolvePools() {
	cachePath := filepath.Join(util.DataDirectory(), "pools.json")
	manifest := loadPoolManifest(poolListURL, cachePath, fetchPoolList)
	payouts.SetConfigs(manifest.Payouts)
	pools.SetConfigs(manifest.Pools)
}

// loadPoolManifest fetches pools.json and caches it at cachePath. If the fetch
// fails or yields no usable lists, the cached copy is used, and failing that
// the built-in lists.
func loadPoolManifest(listURL, cachePath string, fetch func(string, interface{}) error) poolManifest {
	fetched := poolManifest{}
	err := fetch(listURL, &fetched)
	if err == nil {
		if valid, ok := validPoolManifest(fetched); ok {
			if err := writePoolCache(cachePath, valid); err != nil {
				logging.Warnf("Unable to cache the pool list: %v\n", err)
			}
			return valid
		}
		logging.Warnf("The fetched pool list is not usable\n")
	} else {
		logging.Warnf("Unable to fetch the pool list: %v\n", err)
	}

	if data, err := os.ReadFile(cachePath); err == nil {
		cached := poolManifest{}
		if err := json.Unmarshal(data, &cached); err == nil {
			if valid, ok := validPoolManifest(cached); ok {
				logging.Infof("Using the cached pool list\n")
				return valid
			}
		}
	}

	logging.Infof("Using the built-in pool list\n")
	return poolManifest{
		Payouts: append([]payouts.Config(nil), payouts.DefaultConfigs...),
		Pools:   append([]pools.Config(nil), pools.DefaultConfigs...),
	}
}

// validPoolManifest drops invalid payouts and pools. It is usable only when the
// payout to the built-in Dogecoin wallet is kept and at least one pool is left.
func validPoolManifest(m poolManifest) (poolManifest, bool) {
	valid := poolManifest{Payouts: validPayoutConfigs(m.Payouts)}
	tickers := map[string]bool{}
	hasDOGE := false
	for _, c := range valid.Payouts {
		tickers[c.Ticker] = true
		hasDOGE = hasDOGE || (c.ID == payouts.DOGEID && c.Ticker == "DOGE")
	}
	if !hasDOGE {
		logging.Warnf("The pool list has no DOGE payout with ID %d\n", payouts.DOGEID)
		return poolManifest{}, false
	}
	valid.Pools = validPoolConfigs(m.Pools, tickers)
	return valid, len(valid.Pools) > 0
}

var tickerPattern = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// validPayoutConfigs drops invalid entries and repeated IDs or tickers.
func validPayoutConfigs(configs []payouts.Config) []payouts.Config {
	valid := make([]payouts.Config, 0, len(configs))
	seenIDs := map[int]bool{}
	seenTickers := map[string]bool{}
	for _, c := range configs {
		reason := ""
		switch {
		case c.ID < 1:
			reason = "ID must be 1 or higher"
		case !tickerPattern.MatchString(c.Ticker):
			reason = "invalid ticker"
		case strings.TrimSpace(c.Name) == "":
			reason = "no name"
		case seenIDs[c.ID]:
			reason = "repeated ID"
		case seenTickers[c.Ticker]:
			reason = "repeated ticker"
		}
		for _, n := range c.Networks {
			if reason == "" && !tickerPattern.MatchString(n) {
				reason = "invalid network " + n
			}
		}
		if reason != "" {
			logging.Warnf("Ignoring payout %d [%s]: %s\n", c.ID, c.Ticker, reason)
			continue
		}
		seenIDs[c.ID] = true
		seenTickers[c.Ticker] = true
		valid = append(valid, c)
	}
	return valid
}

// ID 1 was the testnet P2Proxy pool and ID 2 is P2Pool.
const firstConfiguredPoolID = 3

// validPoolConfigs drops invalid entries and repeated IDs. A pool's payouts
// that are not among tickers are dropped, and every pool must keep DOGE, the
// payout the app falls back to.
func validPoolConfigs(configs []pools.Config, tickers map[string]bool) []pools.Config {
	valid := make([]pools.Config, 0, len(configs))
	seenIDs := map[int]bool{}
	for _, c := range configs {
		known := make([]string, 0, len(c.Payouts))
		for _, t := range c.Payouts {
			if tickers[t] {
				known = append(known, t)
			} else {
				logging.Warnf("Pool %d [%s]: ignoring unknown payout %s\n", c.ID, c.Name, t)
			}
		}
		c.Payouts = known
		if reason := invalidPoolConfig(c); reason != "" {
			logging.Warnf("Ignoring pool %d [%s]: %s\n", c.ID, c.Name, reason)
			continue
		}
		if seenIDs[c.ID] {
			logging.Warnf("Ignoring pool %d [%s]: repeated ID\n", c.ID, c.Name)
			continue
		}
		seenIDs[c.ID] = true
		valid = append(valid, c)
	}
	return valid
}

// invalidPoolConfig says what is wrong with an entry, or returns "" when it is
// usable.
func invalidPoolConfig(c pools.Config) string {
	switch {
	case c.ID < firstConfiguredPoolID:
		return "ID must be 3 or higher"
	case strings.TrimSpace(c.Name) == "":
		return "no name"
	case !validStratumURL(c.StratumURL):
		return "invalid stratum URL"
	case c.Password == "":
		return "no password"
	case c.Fee == nil || *c.Fee < 0 || *c.Fee >= 100:
		return "missing or invalid fee"
	case !containsTicker(c.Payouts, "DOGE"):
		return "no DOGE payout"
	case c.PayoutURL != "" && !validPoolWebURL(c.PayoutURL):
		return "invalid payout URL"
	case c.PayoutURL != "" && len(c.PayoutFields) == 0:
		return "payout URL without payout fields"
	case c.DashboardURL != "" && !validPoolWebURL(c.DashboardURL):
		return "invalid dashboard URL"
	case c.ActivationURL != "" && !validPoolWebURL(c.ActivationURL):
		return "invalid activation URL"
	case c.ShortageURL != "" && !validPoolWebURL(c.ShortageURL):
		return "invalid shortage URL"
	}
	for _, f := range c.PayoutFields {
		if f == "" || strings.HasPrefix(f, ".") || strings.HasSuffix(f, ".") || strings.Contains(f, "..") {
			return "invalid payout field " + f
		}
	}
	return ""
}

func containsTicker(tickers []string, ticker string) bool {
	for _, t := range tickers {
		if t == ticker {
			return true
		}
	}
	return false
}

// validStratumURL accepts stratum+tcp://host:port with nothing after the port.
func validStratumURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "stratum+tcp" && u.Hostname() != "" && u.Port() != "" &&
		u.User == nil && (u.Path == "" || u.Path == "/") && u.RawQuery == "" && u.Fragment == ""
}

// validPoolWebURL accepts https:// URLs with a host and no credentials, once
// {address} is filled in.
func validPoolWebURL(s string) bool {
	u, err := url.Parse(strings.ReplaceAll(s, "{address}", "address"))
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func writePoolCache(cachePath string, m poolManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	tmp := cachePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, cachePath); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
