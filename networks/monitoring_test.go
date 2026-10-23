package networks

import (
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
	SetNetwork(false)

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
