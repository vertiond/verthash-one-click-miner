package backend

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/networks"
	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/util"
)

// endpointsURL is published next to miners.json, so the third-party servers
// can change without a release.
const endpointsURL = "https://raw.githubusercontent.com/vertiond/verthash-one-click-miner/master/backends.json"

type endpointList struct {
	BlockbookURLs  []string          `json:"blockbookUrls,omitempty"`
	CoinpaprikaURL string            `json:"coinpaprikaUrl,omitempty"`
	CoinpaprikaIDs map[string]string `json:"coinpaprikaIds,omitempty"`
}

func (l endpointList) empty() bool {
	return len(l.BlockbookURLs) == 0 && l.CoinpaprikaURL == "" && len(l.CoinpaprikaIDs) == 0
}

// ResolveEndpointsInBackground loads the Blockbook servers and the CoinPaprika
// endpoint from backends.json (or its cached copy) without blocking the
// caller. Blockbook and price lookups wait until it is done. Run it after
// networks.SetNetwork.
func (m *Backend) ResolveEndpointsInBackground() {
	release := networks.HoldEndpoints()
	go func() {
		defer release()
		cachePath := filepath.Join(util.DataDirectory(), "backends.json")
		l := loadEndpoints(endpointsURL, cachePath, util.GetJson)
		networks.SetBlockbookURLs(l.BlockbookURLs)
		payouts.SetCoinpaprika(l.CoinpaprikaURL, l.CoinpaprikaIDs)
	}()
}

// loadEndpoints fetches the endpoint list and caches it at cachePath. If the
// fetch fails or yields nothing valid, the cached copy is used. Fields that
// end up empty keep the built-in defaults when applied.
func loadEndpoints(listURL, cachePath string, fetch func(string, interface{}) error) endpointList {
	fetched := endpointList{}
	err := fetch(listURL, &fetched)
	if err == nil {
		if l := validEndpoints(fetched); !l.empty() {
			if err := writeEndpointCache(cachePath, l); err != nil {
				logging.Warnf("Unable to cache the endpoint list: %v\n", err)
			}
			return l
		}
		logging.Warnf("The fetched endpoint list has no valid entries\n")
	} else {
		logging.Warnf("Unable to fetch the endpoint list: %v\n", err)
	}

	if data, err := os.ReadFile(cachePath); err == nil {
		cached := endpointList{}
		if err := json.Unmarshal(data, &cached); err == nil {
			if l := validEndpoints(cached); !l.empty() {
				logging.Infof("Using the cached endpoint list\n")
				return l
			}
		}
	}

	logging.Infof("Using the built-in endpoints\n")
	return endpointList{}
}

// validEndpoints drops every invalid entry. Blockbook URLs get a trailing
// "/", the CoinPaprika URL loses it, as the code using them expects.
func validEndpoints(l endpointList) endpointList {
	valid := endpointList{}
	for _, s := range l.BlockbookURLs {
		if !validBaseURL(s) {
			logging.Warnf("Ignoring invalid Blockbook server [%s]\n", s)
			continue
		}
		valid.BlockbookURLs = append(valid.BlockbookURLs, strings.TrimRight(s, "/")+"/")
	}
	if l.CoinpaprikaURL != "" {
		if validBaseURL(l.CoinpaprikaURL) {
			valid.CoinpaprikaURL = strings.TrimRight(l.CoinpaprikaURL, "/")
		} else {
			logging.Warnf("Ignoring invalid CoinPaprika URL [%s]\n", l.CoinpaprikaURL)
		}
	}
	for ticker, id := range l.CoinpaprikaIDs {
		if ticker == "" || !coinpaprikaIDPattern.MatchString(id) {
			logging.Warnf("Ignoring invalid CoinPaprika id [%s: %s]\n", ticker, id)
			continue
		}
		if valid.CoinpaprikaIDs == nil {
			valid.CoinpaprikaIDs = map[string]string{}
		}
		valid.CoinpaprikaIDs[ticker] = id
	}
	return valid
}

// coinpaprikaIDPattern matches ids like "doge-dogecoin". They end up in a URL
// path, so nothing else is allowed.
var coinpaprikaIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validBaseURL accepts https:// URLs with a host and no credentials, query or
// fragment.
func validBaseURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && strings.HasPrefix(s, "https://") && u.Host != "" &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && !u.ForceQuery
}

func writeEndpointCache(cachePath string, l endpointList) error {
	data, err := json.MarshalIndent(l, "", "  ")
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
