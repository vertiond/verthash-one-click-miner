package main

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// OCM's own node checks (ping/ping.go). The report flags nodes that would
// fail them but still lists them; the reviewer decides.
const (
	maxFee    = 2.0
	maxMiners = 35
)

// dynamicDNS lists dynamic-DNS domains. Their registration date belongs to
// the DNS provider, not to whoever runs the node.
var dynamicDNS = []string{
	"ddns.net", "webhop.me", "hopto.org", "zapto.org", "sytes.net", "no-ip.org", "no-ip.biz",
	"no-ip.info", "duckdns.org", "dynu.net", "freeddns.org", "mooo.com", "servebeer.com",
	"serveftp.com", "myftp.org", "mine.nu", "dyndns.org", "dynv6.net",
}

func isDynamicDNS(host string) bool {
	for _, d := range dynamicDNS {
		if strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// flags lists the OCM checks a node would fail.
func flags(s Stats) []string {
	var out []string
	if fee := s.Fee + s.DonationProportion; fee > maxFee {
		out = append(out, fmt.Sprintf("fee + donation %s%% is above %g%%", trim(fee), maxFee))
	}
	if n := len(s.MinerHashRates); n > maxMiners {
		out = append(out, fmt.Sprintf("%d miners is above %d", n, maxMiners))
	}
	if s.Synced != nil && !*s.Synced {
		out = append(out, "reports synced: false")
	}
	return out
}

// trim formats f with at most two decimals and no trailing zeros.
func trim(f float64) string {
	return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64)
}

// body is the pull request description for the changes to both lists.
// A node added to or removed from both lists is described once.
func body(legacy, v2 Plan, r RDAP, now time.Time, v2Created bool) string {
	var b strings.Builder
	b.WriteString("Proposed changes to OCM's P2Pool node lists, from a scan on " + now.Format("2006-01-02") + ".\n\n")
	b.WriteString("`p2pool_nodes.json` is read by OCM releases without the payout check and only lists legacy nodes. ")
	b.WriteString("`p2pool_nodes_v2.json` also lists p2pool-go nodes. Nodes found by the scan are listed under their IP; ")
	b.WriteString("swap in a domain before merging if you know one.\n")

	added, addedIn := mergeAdded(legacy.Added, v2.Added)
	if len(added) > 0 {
		fmt.Fprintf(&b, "\n## Added (%d)\n", len(added))
		for _, n := range added {
			fmt.Fprintf(&b, "\n### %s\n\n", n.IP)
			fmt.Fprintf(&b, "Lists: %s\n\n", addedIn[n.IP])
			b.WriteString("| Network | Fee + donation (%) | Miners | Uptime | Version | Protocol | Synced | Response |\n")
			b.WriteString("|---|---|---|---|---|---|---|---|\n")
			fmt.Fprintf(&b, "| %s | %.2f | %d | %.1f days | %s | %d | %s | %d ms |\n",
				n.Network, n.Stats.Fee+n.Stats.DonationProportion, len(n.Stats.MinerHashRates),
				n.Stats.Uptime/86400, remoteText(n.Stats.Version), n.Stats.ProtocolVersion, synced(n.Stats.Synced), n.ResponseMs)
			for _, f := range flags(n.Stats) {
				fmt.Fprintf(&b, "\n**%s**\n", f)
			}
			fmt.Fprintf(&b, "\nWHOIS: %s\n", whois(entryFor(n).Hostname, r, now))
		}
	}

	removed, removedIn := mergeRemoved(legacy.Removed, v2.Removed)
	if len(removed) > 0 {
		fmt.Fprintf(&b, "\n## Removed (%d)\n", len(removed))
		for _, e := range removed {
			fmt.Fprintf(&b, "\n### %s\n\n", e.Entry.Hostname)
			fmt.Fprintf(&b, "Lists: %s\n\n", removedIn[e.Entry.URL])
			fmt.Fprintf(&b, "Didn't answer at %s: %s\n", e.Entry.URL, remoteText(e.Err))
			fmt.Fprintf(&b, "\nWHOIS: %s\n", whois(hostOf(e.Entry.URL), r, now))
		}
	}

	if v2Created {
		b.WriteString("\nCreates `p2pool_nodes_v2.json` from `p2pool_nodes.json`. OCM releases with the payout check read it.\n")
	} else if len(added) == 0 && len(removed) == 0 {
		b.WriteString("\nNo changes.\n")
	}
	return b.String()
}

// maxRemoteText caps text a remote node controls, such as its version.
const maxRemoteText = 120

// remoteText makes text a remote node controls safe to show: one line, at
// most maxRemoteText characters, in a code span so it isn't formatted, with
// table pipes escaped and @ broken up so it can't mention anyone.
func remoteText(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r):
			return ' '
		case r == '`':
			return '\''
		}
		return r
	}, s)
	if r := []rune(s); len(r) > maxRemoteText {
		s = string(r[:maxRemoteText]) + "..."
	}
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "@", "@"+zeroWidthSpace)
	return "`" + s + "`"
}

// zeroWidthSpace after @ stops GitHub from treating it as a mention.
var zeroWidthSpace = string(rune(0x200B))

func synced(s *bool) string {
	switch {
	case s == nil:
		return "not reported"
	case *s:
		return "yes"
	default:
		return "no"
	}
}

// whois describes who registered host: a domain's registrar and age, or an
// IP's network owner. Lookup failures are shown, never fatal.
func whois(host string, r RDAP, now time.Time) string {
	if net.ParseIP(host) != nil {
		network, owner, country, err := r.IP(host)
		if err != nil {
			return "RDAP lookup failed: " + err.Error()
		}
		var parts []string
		for _, p := range [][2]string{{"Network", network}, {"owner", owner}, {"country", country}} {
			if p[1] != "" {
				parts = append(parts, p[0]+" "+p[1])
			}
		}
		if len(parts) == 0 {
			return "RDAP lookup found no network details"
		}
		return strings.Join(parts, ", ")
	}
	if isDynamicDNS(host) {
		return "dynamic DNS, age not meaningful"
	}
	registrar, registered, err := r.Domain(host)
	if err != nil {
		return "RDAP lookup failed: " + err.Error()
	}
	years := int(now.Sub(registered).Hours() / 24 / 365.25)
	return fmt.Sprintf("Registrar %s, registered %s (%d years ago)", registrar, registered.Format("2006-01-02"), years)
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

// lists names the list files a node is in, given whether it is in each.
func lists(inLegacy, inV2 bool) string {
	switch {
	case inLegacy && inV2:
		return "p2pool_nodes.json, p2pool_nodes_v2.json"
	case inLegacy:
		return "p2pool_nodes.json only"
	default:
		return "p2pool_nodes_v2.json only"
	}
}

func mergeAdded(legacy, v2 []ScanNode) ([]ScanNode, map[string]string) {
	inLegacy, inV2 := map[string]bool{}, map[string]bool{}
	var all []ScanNode
	for _, n := range legacy {
		inLegacy[n.IP] = true
		all = append(all, n)
	}
	for _, n := range v2 {
		inV2[n.IP] = true
		if !inLegacy[n.IP] {
			all = append(all, n)
		}
	}
	where := map[string]string{}
	for _, n := range all {
		where[n.IP] = lists(inLegacy[n.IP], inV2[n.IP])
	}
	return all, where
}

func mergeRemoved(legacy, v2 []Removed) ([]Removed, map[string]string) {
	inLegacy, inV2 := map[string]bool{}, map[string]bool{}
	var all []Removed
	for _, e := range legacy {
		inLegacy[e.Entry.URL] = true
		all = append(all, e)
	}
	for _, e := range v2 {
		inV2[e.Entry.URL] = true
		if !inLegacy[e.Entry.URL] {
			all = append(all, e)
		}
	}
	where := map[string]string{}
	for _, e := range all {
		where[e.Entry.URL] = lists(inLegacy[e.Entry.URL], inV2[e.Entry.URL])
	}
	return all, where
}
