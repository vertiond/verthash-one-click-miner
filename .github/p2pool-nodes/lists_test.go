package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fakeResolve(m map[string][]string) Resolve {
	return func(host string) []string { return m[host] }
}

// fakeProbe fails for the URLs in fail and answers for every other URL.
func fakeProbe(fail map[string]error) Probe {
	return func(url string) error { return fail[url] }
}

func legacyNode(ipPort string) ScanNode { return ScanNode{IP: ipPort, Network: "legacy"} }

var legacyOnly = map[string]bool{"legacy": true}

func TestPlanKeepsMatchedEntry(t *testing.T) {
	list := []Entry{{"boofpool.ddns.net", "stratum+tcp://boofpool.ddns.net:9171", "http://boofpool.ddns.net:9171/"}}
	p := plan(list, []ScanNode{legacyNode("1.2.3.4:9171")}, legacyOnly,
		fakeResolve(map[string][]string{"boofpool.ddns.net": {"1.2.3.4"}}), fakeProbe(nil))
	if !reflect.DeepEqual(p.Keep, list) || len(p.Added) != 0 || len(p.Removed) != 0 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestPlanAddsNewNode(t *testing.T) {
	p := plan(nil, []ScanNode{legacyNode("5.6.7.8:9171")}, legacyOnly, fakeResolve(nil), fakeProbe(nil))
	if len(p.Added) != 1 {
		t.Fatalf("plan = %+v", p)
	}
	want := Entry{"5.6.7.8", "stratum+tcp://5.6.7.8:9171", "http://5.6.7.8:9171/"}
	if got := entryFor(p.Added[0]); got != want {
		t.Fatalf("entryFor = %+v, want %+v", got, want)
	}
}

// A node missing from the scan is only removed when probing it directly
// fails, so a crawl cut short can't drop half the list.
func TestPlanRemovalNeedsFailedProbe(t *testing.T) {
	up := Entry{"up.example", "stratum+tcp://up.example:9171", "http://up.example:9171/"}
	down := Entry{"down.example", "stratum+tcp://down.example:9171", "http://down.example:9171/"}
	p := plan([]Entry{up, down}, nil, legacyOnly,
		fakeResolve(map[string][]string{"up.example": {"1.1.1.1"}, "down.example": {"2.2.2.2"}}),
		fakeProbe(map[string]error{down.URL: errors.New("timeout")}))
	if !reflect.DeepEqual(p.Keep, []Entry{up}) {
		t.Fatalf("kept %+v, want only the node that answered", p.Keep)
	}
	if len(p.Removed) != 1 || p.Removed[0].Entry != down || p.Removed[0].Err != "timeout" {
		t.Fatalf("removed %+v", p.Removed)
	}
}

func TestPlanUnresolvableHostname(t *testing.T) {
	gone := Entry{"gone.example", "stratum+tcp://gone.example:9171", "http://gone.example:9171/"}
	p := plan([]Entry{gone}, []ScanNode{legacyNode("1.2.3.4:9171")}, legacyOnly,
		fakeResolve(nil), fakeProbe(map[string]error{gone.URL: errors.New("no such host")}))
	if len(p.Removed) != 1 || len(p.Keep) != 0 || len(p.Added) != 1 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestPlanNetworkFilter(t *testing.T) {
	scan := []ScanNode{{IP: "9.9.9.9:9172", Network: "p2pool-go"}}
	if p := plan(nil, scan, legacyOnly, fakeResolve(nil), fakeProbe(nil)); len(p.Added) != 0 {
		t.Fatalf("legacy list got p2pool-go node: %+v", p.Added)
	}
	p := plan(nil, scan, map[string]bool{"legacy": true, "p2pool-go": true}, fakeResolve(nil), fakeProbe(nil))
	if len(p.Added) != 1 {
		t.Fatalf("v2 list missed the p2pool-go node: %+v", p)
	}
	want := Entry{"9.9.9.9", "stratum+tcp://9.9.9.9:9172", "http://9.9.9.9:9172/"}
	if got := entryFor(p.Added[0]); got != want {
		t.Fatalf("entryFor = %+v, want %+v", got, want)
	}
}

func TestLoadScanRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	for _, body := range []string{"", "[]", "null"} {
		path := filepath.Join(dir, "scan.json")
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := loadScan(path); err == nil {
			t.Errorf("scan %q: want an error", body)
		}
	}
}

func TestLoadScanReadsScannerOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.json")
	body := `[{"ip":"1.2.3.4:9171","domain_name":"","network":"legacy","response_ms":85,
		"node_stats":{"fee":1,"donation_proportion":0.5,"uptime":86400,"miner_hash_rates":{"Va":1},
		"version":"77.0.0","protocol_version":3501,"synced":false}}]`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	scan, err := loadScan(path)
	if err != nil {
		t.Fatal(err)
	}
	s := scan[0].Stats
	if scan[0].IP != "1.2.3.4:9171" || scan[0].ResponseMs != 85 || s.Fee != 1 || s.DonationProportion != 0.5 ||
		len(s.MinerHashRates) != 1 || s.Version != "77.0.0" || s.ProtocolVersion != 3501 || s.Synced == nil || *s.Synced {
		t.Fatalf("decoded %+v", scan[0])
	}
}

// Rewriting the list keeps today's hand-written format byte for byte.
func TestWriteListFormat(t *testing.T) {
	orig, err := os.ReadFile("../../p2pool_nodes.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := readList("../../p2pool_nodes.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := formatList(entries); !bytes.Equal(got, orig) {
		t.Fatalf("formatList changed the file:\n%s", got)
	}
}
