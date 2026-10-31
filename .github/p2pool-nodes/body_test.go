package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func boolPtr(b bool) *bool { return &b }

func TestFlags(t *testing.T) {
	miners := map[string]float64{}
	for i := 0; i < 36; i++ {
		miners[fmt.Sprintf("V%d", i)] = 1
	}
	cases := []struct {
		s    Stats
		want []string
	}{
		{Stats{Fee: 1.5, DonationProportion: 0.6}, []string{"fee + donation 2.1% is above 2%"}},
		{Stats{MinerHashRates: miners}, []string{"36 miners is above 35"}},
		{Stats{Synced: boolPtr(false)}, []string{"reports synced: false"}},
		{Stats{Fee: 1, Synced: boolPtr(true), MinerHashRates: map[string]float64{"Va": 1}}, nil},
	}
	for _, c := range cases {
		if got := flags(c.s); !reflect.DeepEqual(got, c.want) {
			t.Errorf("flags(%+v) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestIsDynamicDNS(t *testing.T) {
	for host, want := range map[string]bool{
		"boofpool.ddns.net":   true,
		"vtc-brn.webhop.me":   true,
		"mindcraftblocks.com": false,
		"ddns.net.example":    false,
		"1.2.3.4":             false,
	} {
		if got := isDynamicDNS(host); got != want {
			t.Errorf("isDynamicDNS(%q) = %v, want %v", host, got, want)
		}
	}
}

// fakeRDAP answers domain and IP lookups like rdap.org, and counts requests.
func fakeRDAP(t *testing.T, status int) (RDAP, *int32) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if status != http.StatusOK {
			http.Error(w, "slow down", status)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/domain/"):
			fmt.Fprint(w, `{"events":[{"eventAction":"registration","eventDate":"2014-03-01T00:00:00Z"},
				{"eventAction":"expiration","eventDate":"2027-03-01T00:00:00Z"}],
				"entities":[{"roles":["registrar"],"vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","Namecheap"]]]}]}`)
		case strings.HasPrefix(r.URL.Path, "/ip/"):
			fmt.Fprint(w, `{"name":"DIGITALOCEAN","country":"US",
				"entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","DigitalOcean, LLC"]]]}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return RDAP{Client: srv.Client(), Base: srv.URL}, &hits
}

func TestRDAPDomain(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	registrar, registered, err := r.Domain("vtc.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if registrar != "Namecheap" || !registered.Equal(time.Date(2014, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %q, %v", registrar, registered)
	}
}

func TestRDAPIP(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	network, owner, country, err := r.IP("1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if network != "DIGITALOCEAN" || owner != "DigitalOcean, LLC" || country != "US" {
		t.Fatalf("got %q, %q, %q", network, owner, country)
	}
}

var now = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

func addedPlan(nodes ...ScanNode) Plan { return Plan{Added: nodes} }

func TestBodyDescribesAddedNode(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	n := ScanNode{IP: "1.2.3.4:9171", Network: "legacy", ResponseMs: 85,
		Stats: Stats{Fee: 1, Uptime: 3 * 86400, MinerHashRates: map[string]float64{"Va": 1}, Version: "77.0.0", ProtocolVersion: 3501}}
	b := body(addedPlan(n), addedPlan(n), r, now, false)
	for _, want := range []string{
		"1.2.3.4:9171", "legacy", "1.00", "| 1 |", "3.0 days", "77.0.0", "3501", "not reported", "85 ms",
		"p2pool_nodes.json, p2pool_nodes_v2.json",
		"Network DIGITALOCEAN, owner DigitalOcean, LLC, country US",
	} {
		if !strings.Contains(b, want) {
			t.Errorf("body is missing %q:\n%s", want, b)
		}
	}
	if strings.Count(b, "### 1.2.3.4:9171") != 1 {
		t.Errorf("a node added to both lists should be described once:\n%s", b)
	}
}

func TestBodyDomainAge(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	gone := Removed{Entry: Entry{"vtc.example.com", "stratum+tcp://vtc.example.com:9171", "http://vtc.example.com:9171/"}, Err: "timeout"}
	b := body(Plan{Removed: []Removed{gone}}, Plan{Removed: []Removed{gone}}, r, now, false)
	for _, want := range []string{"Removed", "vtc.example.com", "timeout", "Registrar Namecheap, registered 2014-03-01 (12 years ago)"} {
		if !strings.Contains(b, want) {
			t.Errorf("body is missing %q:\n%s", want, b)
		}
	}
}

func TestBodyRDAPFailure(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusTooManyRequests)
	n := ScanNode{IP: "1.2.3.4:9171", Network: "legacy"}
	b := body(addedPlan(n), addedPlan(n), r, now, false)
	if !strings.Contains(b, "RDAP lookup failed") || !strings.Contains(b, "1.2.3.4:9171") {
		t.Fatalf("body = %s", b)
	}
}

func TestBodyDynamicDNS(t *testing.T) {
	r, hits := fakeRDAP(t, http.StatusOK)
	gone := Removed{Entry: Entry{"vtc-brn.webhop.me", "stratum+tcp://vtc-brn.webhop.me:9171", "http://vtc-brn.webhop.me:9171/"}, Err: "timeout"}
	b := body(Plan{Removed: []Removed{gone}}, Plan{}, r, now, false)
	if !strings.Contains(b, "dynamic DNS, age not meaningful") {
		t.Fatalf("body = %s", b)
	}
	if *hits != 0 {
		t.Fatalf("made %d RDAP requests for a dynamic DNS host", *hits)
	}
}

func TestBodyFlags(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	n := ScanNode{IP: "9.9.9.9:9172", Network: "p2pool-go", Stats: Stats{Synced: boolPtr(false)}}
	b := body(Plan{}, addedPlan(n), r, now, false)
	if !strings.Contains(b, "**reports synced: false**") || !strings.Contains(b, "p2pool_nodes_v2.json only") {
		t.Fatalf("body = %s", b)
	}
}

// ARIN records often have no country; the line leaves out empty parts.
func TestWhoisIPMissingParts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"INAP-DAL006","entities":[]}`)
	}))
	defer srv.Close()
	got := whois("66.151.242.154", RDAP{Client: srv.Client(), Base: srv.URL}, now)
	if got != "Network INAP-DAL006" {
		t.Fatalf("whois = %q, want %q", got, "Network INAP-DAL006")
	}
}

// Text a remote node controls can't break the table, mention people or
// add formatting to the report reviewers rely on.
func TestBodyEscapesRemoteText(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	evil := "1.0|x\n**verified by maintainers** @vertiond ![x](http://e/x.png)" + strings.Repeat("A", 500)
	n := ScanNode{IP: "1.2.3.4:9171", Network: "legacy", Stats: Stats{Version: evil}}
	gone := Removed{Entry: Entry{"gone.example.com", "stratum+tcp://gone.example.com:9171", "http://gone.example.com:9171/"}, Err: "boom\n@vertiond"}
	b := body(addedPlan(n), Plan{Removed: []Removed{gone}}, r, now, false)
	for _, bad := range []string{"\n**verified", "@vertiond", "1.0|x", strings.Repeat("A", 300)} {
		if strings.Contains(b, bad) {
			t.Errorf("body contains %q:\n%s", bad, b)
		}
	}
	for _, line := range strings.Split(b, "\n") {
		if strings.HasPrefix(line, "| legacy |") && strings.Count(strings.ReplaceAll(line, `\|`, ""), "|") != 9 {
			t.Errorf("table row has extra cells: %s", line)
		}
	}
}

// Until the payout branch adds p2pool_nodes_v2.json, a run creates it; the
// description says so instead of "No changes."
func TestBodyNotesNewV2File(t *testing.T) {
	r, _ := fakeRDAP(t, http.StatusOK)
	b := body(Plan{}, Plan{}, r, now, true)
	if !strings.Contains(b, "Creates `p2pool_nodes_v2.json`") || strings.Contains(b, "No changes.") {
		t.Fatalf("body = %s", b)
	}
}
