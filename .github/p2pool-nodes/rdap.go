package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RDAP looks up domain and IP registration records. RDAP is the JSON
// successor to WHOIS; rdap.org redirects each query to the registry that
// holds the record.
type RDAP struct {
	Client *http.Client
	Base   string // e.g. https://rdap.org
}

type rdapEntity struct {
	Roles      []string      `json:"roles"`
	VcardArray []interface{} `json:"vcardArray"`
}

// Domain returns the registrar and registration date of host's registrable
// domain (its last two labels).
func (r RDAP) Domain(host string) (registrar string, registered time.Time, err error) {
	var rec struct {
		Events []struct {
			Action string    `json:"eventAction"`
			Date   time.Time `json:"eventDate"`
		} `json:"events"`
		Entities []rdapEntity `json:"entities"`
	}
	if err := r.get("domain/"+registrableDomain(host), &rec); err != nil {
		return "", time.Time{}, err
	}
	for _, e := range rec.Events {
		if e.Action == "registration" {
			registered = e.Date
		}
	}
	for _, e := range rec.Entities {
		if hasRole(e, "registrar") {
			registrar = vcardName(e.VcardArray)
		}
	}
	if registered.IsZero() {
		return registrar, registered, fmt.Errorf("no registration date in the RDAP record")
	}
	return registrar, registered, nil
}

// IP returns the name, owner and country of the network an IP belongs to.
func (r RDAP) IP(ip string) (network, owner, country string, err error) {
	var rec struct {
		Name     string       `json:"name"`
		Country  string       `json:"country"`
		Entities []rdapEntity `json:"entities"`
	}
	if err := r.get("ip/"+ip, &rec); err != nil {
		return "", "", "", err
	}
	for _, e := range rec.Entities {
		if name := vcardName(e.VcardArray); name != "" {
			owner = name
			if hasRole(e, "registrant") {
				break
			}
		}
	}
	return rec.Name, owner, rec.Country, nil
}

func (r RDAP) get(path string, into interface{}) error {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSuffix(r.Base, "/")+"/"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/rdap+json")
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RDAP answered %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func registrableDomain(host string) string {
	labels := strings.Split(strings.TrimSuffix(host, "."), ".")
	if len(labels) > 2 {
		labels = labels[len(labels)-2:]
	}
	return strings.Join(labels, ".")
}

func hasRole(e rdapEntity, role string) bool {
	for _, r := range e.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// vcardName returns the fn (formatted name) from a jCard:
// ["vcard", [["fn", {}, "text", "Name"], ...]].
func vcardName(v []interface{}) string {
	if len(v) < 2 {
		return ""
	}
	props, _ := v[1].([]interface{})
	for _, p := range props {
		item, _ := p.([]interface{})
		if len(item) >= 4 && item[0] == "fn" {
			name, _ := item[3].(string)
			return name
		}
	}
	return ""
}
