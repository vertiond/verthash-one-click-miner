package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
)

// Entry is one node in p2pool_nodes.json or p2pool_nodes_v2.json.
type Entry struct {
	Hostname string `json:"Hostname"`
	Stratum  string `json:"Stratum"`
	URL      string `json:"URL"`
}

// ScanNode is one node in p2pool-scanner-go's -once output.
type ScanNode struct {
	IP         string `json:"ip"` // ip:port of the node's web server
	DomainName string `json:"domain_name"`
	Network    string `json:"network"` // "legacy" or "p2pool-go"
	ResponseMs int64  `json:"response_ms"`
	Stats      Stats  `json:"node_stats"`
}

// Stats is the part of a node's /local_stats the report shows.
type Stats struct {
	Fee                float64            `json:"fee"`
	DonationProportion float64            `json:"donation_proportion"`
	Uptime             float64            `json:"uptime"` // seconds
	MinerHashRates     map[string]float64 `json:"miner_hash_rates"`
	Version            string             `json:"version"`
	ProtocolVersion    uint               `json:"protocol_version"`
	Synced             *bool              `json:"synced"` // nil for legacy nodes, which don't report it
}

// Probe checks a node at its list URL; nil means it answered /local_stats.
type Probe func(url string) error

// Resolve returns the IPs a hostname resolves to.
type Resolve func(host string) []string

// Removed is a listed node proposed for removal, with why its probe failed.
type Removed struct {
	Entry Entry
	Err   string
}

// Plan is the proposed change to one list file.
type Plan struct {
	Keep    []Entry    // listed nodes that stay, in list order
	Added   []ScanNode // scanned nodes that aren't listed yet
	Removed []Removed
}

// plan compares a list with a scan. Listed nodes found in the scan stay.
// A listed node missing from the scan stays if probing it directly works and
// is removed if not, so a scan cut short can't drop nodes that are up.
// Scanned nodes of the given networks that aren't listed are added.
func plan(list []Entry, scan []ScanNode, networks map[string]bool, resolve Resolve, probe Probe) Plan {
	scanned := map[string]bool{}
	for _, n := range scan {
		scanned[n.IP] = true
	}

	var p Plan
	matched := map[string]bool{}
	for _, e := range list {
		found := false
		for _, addr := range entryAddrs(e, resolve) {
			if scanned[addr] {
				matched[addr] = true
				found = true
			}
		}
		switch {
		case found:
			p.Keep = append(p.Keep, e)
		case probe(e.URL) == nil:
			p.Keep = append(p.Keep, e)
		default:
			p.Removed = append(p.Removed, Removed{Entry: e, Err: probe(e.URL).Error()})
		}
	}

	for _, n := range scan {
		if networks[n.Network] && !matched[n.IP] {
			p.Added = append(p.Added, n)
		}
	}
	return p
}

// entryAddrs returns the ip:port addresses a list entry's URL points at.
func entryAddrs(e Entry, resolve Resolve) []string {
	u, err := url.Parse(e.URL)
	if err != nil {
		return nil
	}
	host, port := u.Hostname(), u.Port()
	ips := []string{host}
	if net.ParseIP(host) == nil {
		ips = resolve(host)
	}
	var addrs []string
	for _, ip := range ips {
		addrs = append(addrs, net.JoinHostPort(ip, port))
	}
	return addrs
}

// entryFor is the list entry for a newly found node, under its IP.
func entryFor(n ScanNode) Entry {
	host, _, err := net.SplitHostPort(n.IP)
	if err != nil {
		host = n.IP
	}
	return Entry{Hostname: host, Stratum: "stratum+tcp://" + n.IP, URL: "http://" + n.IP + "/"}
}

// loadScan reads the scanner's output. An empty scan is an error: proposing
// changes from it would only be noise.
func loadScan(path string) ([]ScanNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scan []ScanNode
	if err := json.Unmarshal(data, &scan); err != nil {
		return nil, err
	}
	if len(scan) == 0 {
		return nil, errors.New("the scan found no nodes")
	}
	return scan, nil
}

func readList(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list []Entry
	return list, json.Unmarshal(data, &list)
}

// formatList writes a list in the same hand-aligned layout as today's file.
func formatList(list []Entry) []byte {
	var b bytes.Buffer
	b.WriteString("[\n")
	for i, e := range list {
		b.WriteString("\t{\n")
		b.WriteString("\t\t\"Hostname\": " + quote(e.Hostname) + ",\n")
		b.WriteString("\t\t\"Stratum\":  " + quote(e.Stratum) + ",\n")
		b.WriteString("\t\t\"URL\":      " + quote(e.URL) + "\n")
		b.WriteString("\t}")
		if i < len(list)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return b.Bytes()
}

// quote JSON-encodes s without escaping &, < and >.
func quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return string(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
}
