package p2poolgo

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Mode is how the p2pool-go node gets its blockchain data.
type Mode int

const (
	// ModeLight runs the node without a local Vertcoin Core.
	ModeLight Mode = iota
	// ModeFull runs the node against a synced local Vertcoin Core.
	ModeFull
)

const defaultCoreRPCPort = 5888

// Detection is the result of looking for a local Vertcoin Core.
type Detection struct {
	Mode       Mode
	CookiePath string // full mode only, when the cookie was used
	RPCUser    string // full mode only, when vertcoin.conf's rpcuser/rpcpassword were used
	RPCPass    string // never log it
	RPCPort    int    // from rpcport= or 5888
	RPCNotice  bool   // Core running but RPC unusable
	ConfPath   string // absolute vertcoin.conf path, for the notice

	// CoreSyncing: Core's RPC works but Core isn't ready to mine through yet:
	// it is in initial block download, still loading at startup, or too busy
	// syncing to answer in time. SyncProgress is its verificationprogress
	// (0 to 1), or -1 when unknown.
	CoreSyncing  bool
	SyncProgress float64
}

// coreRPCTimeout bounds the getblockchaininfo probe. Injectable for tests.
var coreRPCTimeout = 5 * time.Second

// coreState is what a getblockchaininfo probe found.
type coreState int

const (
	coreUnusable coreState = iota // can't be queried with these credentials
	coreSynced
	coreSyncing // initial block download, warming up, or too busy to answer
)

// CoreEnv holds the injectable inputs of DetectCore.
type CoreEnv struct {
	DataDir        string      // CoreDataDir() in production
	ProcessRunning func() bool // coreProcessRunning in production
	RPCHost        string      // "127.0.0.1" in production
}

// ProductionCoreEnv is the CoreEnv for detecting the real local Vertcoin Core.
func ProductionCoreEnv() CoreEnv {
	return CoreEnv{DataDir: CoreDataDir(), ProcessRunning: coreProcessRunning, RPCHost: "127.0.0.1"}
}

// CoreDataDir is the default Vertcoin Core data directory.
func CoreDataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "Vertcoin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".vertcoin")
}

// parseConf returns the first value per key, ignoring comments and sections.
func parseConf(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if _, seen := out[k]; !seen {
			out[k] = v
		}
	}
	return out
}

// DetectCore decides between full and light mode (spec 5.4).
func DetectCore(env CoreEnv) Detection {
	d := Detection{Mode: ModeLight, RPCPort: defaultCoreRPCPort}
	d.ConfPath = filepath.Join(env.DataDir, "vertcoin.conf")
	if abs, err := filepath.Abs(d.ConfPath); err == nil {
		d.ConfPath = abs
	}
	conf := parseConf(d.ConfPath)
	cookieDir := env.DataDir
	if v := conf["datadir"]; v != "" {
		cookieDir = v
	}
	if v, err := strconv.Atoi(conf["rpcport"]); err == nil && v > 0 && v < 65536 {
		d.RPCPort = v
	}
	host := env.RPCHost
	if host == "" {
		host = "127.0.0.1"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(d.RPCPort))

	running := env.ProcessRunning != nil && env.ProcessRunning()
	if !running {
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			c.Close()
			running = true
		}
	}
	if !running {
		return d
	}

	d.RPCNotice = true // until proven usable
	// Core writes no cookie when rpcpassword is set, so use the conf's own
	// credentials then. rpcauth= is hashed and can't be used; it falls
	// through to the cookie like no credentials at all.
	if user, pass := conf["rpcuser"], conf["rpcpassword"]; user != "" && pass != "" {
		if d.applyCoreState(probeCore(addr, user, pass)) {
			d.RPCUser, d.RPCPass = user, pass
		}
		return d
	}
	cookiePath := filepath.Join(cookieDir, ".cookie")
	cookie, err := os.ReadFile(cookiePath)
	if err != nil {
		return d
	}
	user, pass, ok := strings.Cut(strings.TrimSpace(string(cookie)), ":")
	if !ok {
		return d
	}
	if d.applyCoreState(probeCore(addr, user, pass)) {
		d.CookiePath = cookiePath
	}
	return d
}

// applyCoreState records a probe's result and reports whether Core can be
// mined through now, in which case the caller records the credentials used.
func (d *Detection) applyCoreState(state coreState, progress float64) bool {
	switch state {
	case coreSynced:
		d.RPCNotice = false
		d.Mode = ModeFull
		return true
	case coreSyncing:
		d.RPCNotice = false
		d.CoreSyncing = true
		d.SyncProgress = progress
	}
	return false
}

// probeCore calls getblockchaininfo with basic auth. progress is Core's
// verificationprogress, or -1 when it didn't say.
func probeCore(addr, user, pass string) (state coreState, progress float64) {
	req, err := http.NewRequest("POST", "http://"+addr+"/", strings.NewReader(
		`{"jsonrpc":"1.0","id":"ocm","method":"getblockchaininfo","params":[]}`))
	if err != nil {
		return coreUnusable, -1
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: coreRPCTimeout}).Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			// The port answered but Core didn't reply in time: it is busy,
			// which during initial block download is normal.
			return coreSyncing, -1
		}
		return coreUnusable, -1
	}
	defer resp.Body.Close()
	var body struct {
		Result *struct {
			IBD      bool    `json:"initialblockdownload"`
			Progress float64 `json:"verificationprogress"`
		} `json:"result"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	decodeErr := json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode != http.StatusOK {
		// RPC_IN_WARMUP: Core is loading its block index or verifying blocks.
		if decodeErr == nil && body.Error != nil && body.Error.Code == -28 {
			return coreSyncing, -1
		}
		return coreUnusable, -1
	}
	if decodeErr != nil || body.Result == nil {
		return coreUnusable, -1
	}
	if body.Result.IBD {
		return coreSyncing, body.Result.Progress
	}
	return coreSynced, body.Result.Progress
}
