package p2poolgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// StratumPort is the local stratum (and HTTP API) port of the node.
	StratumPort = 9172
	// P2PPort is the node's p2p port.
	P2PPort = 9348
)

// LocalURL is the base URL of a node on this machine.
func LocalURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", StratumPort)
}

// Status is the subset of the node's /api/status that OCM uses.
type Status struct {
	Ready              bool    `json:"ready"`
	State              string  `json:"state"` // waiting_rpc | waiting_template | syncing_sharechain | ready
	Version            string  `json:"version"`
	Peers              int     `json:"peers"`
	Height             int64   `json:"height"`
	Difficulty         float64 `json:"difficulty"`
	ExpectedPayoutSats int64   `json:"expected_payout_sats"`
	UptimeSeconds      int64   `json:"uptime_seconds"`
}

var ErrNotP2PoolGo = errors.New("not a p2pool-go node")

var statusClient = &http.Client{Timeout: 5 * time.Second}

// GetStatus queries baseURL/api/status. A 200 response lacking version and
// state is not a p2pool-go node.
func GetStatus(baseURL, address string) (*Status, error) {
	resp, err := statusClient.Get(baseURL + "/api/status?address=" + url.QueryEscape(address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status endpoint returned %s", resp.Status)
	}
	var s Status
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, ErrNotP2PoolGo
	}
	if s.Version == "" || s.State == "" {
		return nil, ErrNotP2PoolGo
	}
	return &s, nil
}

// DetectExisting polls twice, gap apart; true only if both succeed and the
// uptime increased (so a node that is shutting down is not adopted).
func DetectExisting(baseURL string, gap time.Duration) bool {
	a, err := GetStatus(baseURL, "")
	if err != nil {
		return false
	}
	time.Sleep(gap)
	b, err := GetStatus(baseURL, "")
	if err != nil {
		return false
	}
	return b.UptimeSeconds > a.UptimeSeconds
}
