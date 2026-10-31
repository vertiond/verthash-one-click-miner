// Command p2pool-nodes turns a p2pool-scanner-go scan into proposed changes
// to OCM's P2Pool node lists and a pull request description.
//
// p2pool_nodes.json is read by OCM releases without the payout check, so it
// only ever lists legacy nodes. p2pool_nodes_v2.json also lists p2pool-go
// nodes.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	scanPath := flag.String("scan", "", "scanner -once output")
	legacyPath := flag.String("legacy", "p2pool_nodes.json", "legacy-only node list")
	v2Path := flag.String("v2", "p2pool_nodes_v2.json", "node list with every network")
	bodyPath := flag.String("body", "pr-body.md", "where to write the pull request description")
	flag.Parse()

	if err := run(*scanPath, *legacyPath, *v2Path, *bodyPath); err != nil {
		log.Fatal(err)
	}
}

func run(scanPath, legacyPath, v2Path, bodyPath string) error {
	scan, err := loadScan(scanPath)
	if err != nil {
		return fmt.Errorf("reading the scan: %w", err)
	}
	legacyOrig, err := os.ReadFile(legacyPath)
	if err != nil {
		return err
	}
	legacy, err := readList(legacyPath)
	if err != nil {
		return err
	}
	v2Orig, err := os.ReadFile(v2Path)
	if errors.Is(err, os.ErrNotExist) {
		v2Orig = nil // start the v2 list from the legacy one
	} else if err != nil {
		return err
	}
	v2 := legacy
	if v2Orig != nil {
		if v2, err = readList(v2Path); err != nil {
			return err
		}
	}

	resolve, probe := cachedResolve(), cachedProbe()
	legacyPlan := plan(legacy, scan, map[string]bool{"legacy": true}, resolve, probe)
	v2Plan := plan(v2, scan, map[string]bool{"legacy": true, "p2pool-go": true}, resolve, probe)

	legacyNew := formatList(withAdded(legacyPlan))
	v2New := formatList(withAdded(v2Plan))
	changed := !bytes.Equal(legacyNew, legacyOrig) || !bytes.Equal(v2New, v2Orig)
	if changed {
		if err := os.WriteFile(legacyPath, legacyNew, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(v2Path, v2New, 0644); err != nil {
			return err
		}
	}
	rdap := RDAP{Client: &http.Client{Timeout: 10 * time.Second}, Base: "https://rdap.org"}
	if err := os.WriteFile(bodyPath, []byte(body(legacyPlan, v2Plan, rdap, time.Now().UTC(), v2Orig == nil)), 0644); err != nil {
		return err
	}
	log.Printf("legacy list: %d added, %d removed; v2 list: %d added, %d removed",
		len(legacyPlan.Added), len(legacyPlan.Removed), len(v2Plan.Added), len(v2Plan.Removed))
	return setOutput("changed", fmt.Sprint(changed))
}

// withAdded is the new list: the nodes that stay, then the new ones.
func withAdded(p Plan) []Entry {
	list := append([]Entry(nil), p.Keep...)
	for _, n := range p.Added {
		list = append(list, entryFor(n))
	}
	return list
}

// setOutput records a step output when running in GitHub Actions.
func setOutput(name, value string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
	return err
}

// cachedResolve looks up each hostname once.
func cachedResolve() Resolve {
	var mu sync.Mutex
	cache := map[string][]string{}
	return func(host string) []string {
		mu.Lock()
		defer mu.Unlock()
		if ips, ok := cache[host]; ok {
			return ips
		}
		ips, _ := net.LookupHost(host)
		cache[host] = ips
		return ips
	}
}

// cachedProbe probes each URL once: GET <url>local_stats must answer with a
// JSON object.
func cachedProbe() Probe {
	client := &http.Client{Timeout: 10 * time.Second}
	var mu sync.Mutex
	cache := map[string]error{}
	return func(url string) error {
		mu.Lock()
		defer mu.Unlock()
		if err, ok := cache[url]; ok {
			return err
		}
		err := probeOnce(client, url)
		cache[url] = err
		return err
	}
}

func probeOnce(client *http.Client, url string) error {
	r, err := client.Get(url + "local_stats")
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("local_stats answered %s", r.Status)
	}
	var stats map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&stats); err != nil || stats == nil {
		return fmt.Errorf("local_stats is not a JSON object")
	}
	return nil
}
