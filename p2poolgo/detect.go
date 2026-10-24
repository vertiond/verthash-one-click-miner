package p2poolgo

import (
	"bufio"
	"encoding/json"
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
}

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
		ibd, ok := coreIBD(addr, user, pass)
		if !ok {
			return d
		}
		d.RPCNotice = false
		if !ibd {
			d.Mode = ModeFull
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
	ibd, ok := coreIBD(addr, user, pass)
	if !ok {
		return d
	}
	d.RPCNotice = false
	if !ibd {
		d.Mode = ModeFull
		d.CookiePath = cookiePath
	}
	return d
}

// coreIBD calls getblockchaininfo with basic auth. ok is false when Core
// can't be queried with these credentials.
func coreIBD(addr, user, pass string) (ibd, ok bool) {
	req, err := http.NewRequest("POST", "http://"+addr+"/", strings.NewReader(
		`{"jsonrpc":"1.0","id":"ocm","method":"getblockchaininfo","params":[]}`))
	if err != nil {
		return false, false
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, false
	}
	var body struct {
		Result *struct {
			IBD bool `json:"initialblockdownload"`
		} `json:"result"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil || body.Result == nil {
		return false, false
	}
	return body.Result.IBD, true
}
