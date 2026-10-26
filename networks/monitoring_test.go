package networks

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The monitoring worker checks the servers listed in its wrangler.toml, so those
// lists have to match the ones the app really uses.
func workerList(t *testing.T, toml, name string) []string {
	t.Helper()
	// \r?: Git on Windows checks the file out with CRLF line endings
	m := regexp.MustCompile(`(?m)^` + name + ` = "(.*)"\r?$`).FindStringSubmatch(toml)
	if m == nil {
		t.Fatalf("%s not found in monitoring/worker/wrangler.toml", name)
	}
	list := strings.Split(m[1], ",")
	sort.Strings(list)
	return list
}

func TestMonitoringWorkerChecksTheServersTheAppUses(t *testing.T) {
	raw, err := os.ReadFile("../monitoring/worker/wrangler.toml")
	if err != nil {
		t.Fatal(err)
	}
	SetNetwork()

	for name, want := range map[string][]string{
		"BLOCKBOOK_URLS": append([]string{}, Active.BlockbookURLs...),
		"BACKEND_URLS":   append([]string{}, Active.BackendServers...),
	} {
		sort.Strings(want)
		if got := workerList(t, string(raw), name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s in monitoring/worker/wrangler.toml is %v, networks.go has %v", name, got, want)
		}
	}
}

// The app fetches its Blockbook and backend servers from backends.json and
// falls back to the built-in lists, so the worker has to check the same
// servers as both.
func TestMonitoringWorkerChecksThePublishedServers(t *testing.T) {
	raw, err := os.ReadFile("../monitoring/worker/wrangler.toml")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../backends.json")
	if err != nil {
		t.Fatal(err)
	}
	published := struct {
		BlockbookURLs  []string `json:"blockbookUrls"`
		BackendServers []string `json:"backendServers"`
	}{}
	if err := json.Unmarshal(data, &published); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, key string
		published []string
		defaults  []string
	}{
		{"BLOCKBOOK_URLS", "blockbookUrls", published.BlockbookURLs, DefaultBlockbookURLs},
		{"BACKEND_URLS", "backendServers", published.BackendServers, DefaultBackendServers},
	} {
		want := append([]string{}, c.published...)
		sort.Strings(want)
		if got := workerList(t, string(raw), c.name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s in monitoring/worker/wrangler.toml is %v, %s in backends.json has %v", c.name, got, c.key, want)
		}
		defaults := append([]string{}, c.defaults...)
		sort.Strings(defaults)
		if !reflect.DeepEqual(defaults, want) {
			t.Errorf("the built-in list is %v, %s in backends.json has %v", defaults, c.key, want)
		}
	}
}
