package p2poolgo

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
)

const (
	maxBackoff       = 60 * time.Second
	backoffResetUp   = 10 * time.Minute
	rapidFailDeaths  = 5
	rapidFailWindow  = 2 * time.Minute
	gracefulStopWait = 10 * time.Second
	killWait         = 5 * time.Second
	logTailMaxBytes  = 64 * 1024
)

// Config is everything needed to launch OCM's own p2pool-go node.
type Config struct {
	ExePath, DataDir, VerthashPath, Address string
	Detection                               Detection
	TrustedPeers, BootstrapPeers            []string
}

// BuildArgs returns the node's command line. The order is fixed.
func BuildArgs(cfg Config, parentPID int, token string) []string {
	return buildArgs(cfg, parentPID, token, StratumPort)
}

func buildArgs(cfg Config, parentPID int, token string, stratumPort int) []string {
	args := []string{
		"-datadir", cfg.DataDir,
		"-verthash", cfg.VerthashPath,
		"-bind", "127.0.0.1",
		"-stratum", strconv.Itoa(stratumPort),
		"-parentpid", strconv.Itoa(parentPID),
		"-shutdown-token", token,
		"-pooladdr", cfg.Address,
	}
	if len(cfg.TrustedPeers) > 0 {
		args = append(args, "-trustedpeers", strings.Join(cfg.TrustedPeers, ","))
	}
	if len(cfg.BootstrapPeers) > 0 {
		args = append(args, "-peers", strings.Join(cfg.BootstrapPeers, ","))
	}
	if cfg.Detection.Mode == ModeFull {
		if cfg.Detection.CookiePath == "" && cfg.Detection.RPCUser != "" {
			// vertcoin.conf's own credentials; already plain text there.
			args = append(args, "-u", cfg.Detection.RPCUser, "-p", cfg.Detection.RPCPass)
		} else {
			args = append(args, "-cookie", cfg.Detection.CookiePath)
		}
		if p := cfg.Detection.RPCPort; p != 0 && p != defaultCoreRPCPort {
			args = append(args, "-rpcport", strconv.Itoa(p))
		}
	} else {
		args = append(args, "-light", "-p2pbind", "127.0.0.1", "-disable-upnp")
	}
	return args
}

// NextBackoff is the delay before restart number attempt: 2s<<attempt, capped at 60s.
func NextBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt >= 5 {
		return maxBackoff
	}
	d := 2 * time.Second << uint(attempt)
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// deathLog remembers recent unexpected node exits.
type deathLog struct {
	times []time.Time
}

// Record adds a death at t and reports whether there were at least five
// deaths within the two minutes ending at t.
func (d *deathLog) Record(t time.Time) (rapidFail bool) {
	kept := d.times[:0]
	for _, p := range d.times {
		if t.Sub(p) < rapidFailWindow {
			kept = append(kept, p)
		}
	}
	d.times = append(kept, t)
	return len(d.times) >= rapidFailDeaths
}

// PortFree reports whether addr can be listened on right now and nothing
// answers on it. The connect attempt catches a program listening on every
// interface: Windows still lets OCM listen on the loopback address then.
func PortFree(addr string) bool {
	if portAnswers(addr) {
		return false
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// portAnswers reports whether a TCP connect to addr succeeds. A local
// listener accepts at once, so a short timeout is enough.
func portAnswers(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func portInUseError(port int) error {
	return fmt.Errorf("Port %d is already in use by another program. Close it and retry.", port)
}

// Runner launches and supervises OCM's own p2pool-go node.
type Runner struct {
	cfg         Config
	stratumPort int
	p2pPort     int
	argPrefix   []string                // tests only: args placed before the node args
	backoff     func(int) time.Duration // NextBackoff; tests shorten it
	stopWait    time.Duration           // Stop's graceful budget; tests shorten it
	stopHook    func()                  // tests only: runs just before Stop's final state update

	mu          sync.Mutex
	cmd         *exec.Cmd
	exited      chan struct{} // closed when cmd has exited
	token       string
	startedAt   time.Time
	running     bool
	supervising bool
	stopCh      chan struct{} // closed by Stop to end supervision
	supDone     chan struct{} // closed when the supervisor goroutine returns
	rapidFailed bool
	deaths      deathLog
	attempt     int
}

// NewRunner returns a Runner for cfg. Nothing is started.
func NewRunner(cfg Config) *Runner {
	return &Runner{cfg: cfg, stratumPort: StratumPort, p2pPort: P2PPort, backoff: NextBackoff, stopWait: gracefulStopWait}
}

// Config returns the configuration the runner launches the node with.
func (r *Runner) Config() Config {
	return r.cfg
}

// Start checks the node's ports, launches it, and supervises it until Stop
// or a rapid-fail. Starting a runner that is already supervising is a no-op.
func (r *Runner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.supervising {
		return nil
	}
	if !PortFree(net.JoinHostPort("127.0.0.1", strconv.Itoa(r.stratumPort))) {
		return portInUseError(r.stratumPort)
	}
	if !PortFree(r.p2pCheckAddr()) {
		return portInUseError(r.p2pPort)
	}

	r.rapidFailed = false
	r.deaths = deathLog{}
	r.attempt = 0
	if err := r.launchLocked(); err != nil {
		return err
	}
	r.supervising = true
	r.stopCh = make(chan struct{})
	r.supDone = make(chan struct{})
	go r.supervise(r.stopCh, r.supDone)
	return nil
}

// p2pCheckAddr is where OCM test-listens for the node's p2p port. It is
// loopback in every mode: a listen on all interfaces by vertcoin-ocm.exe
// makes Windows show a firewall prompt for OCM itself. In full mode the
// node binds every interface; PortFree's connect attempt still sees a
// program holding the port on any interface.
func (r *Runner) p2pCheckAddr() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(r.p2pPort))
}

// launchLocked starts one node process. r.mu must be held.
func (r *Runner) launchLocked() error {
	token, err := newToken()
	if err != nil {
		return fmt.Errorf("unable to create the node shutdown token: %w", err)
	}
	args := append(append([]string{}, r.argPrefix...), buildArgs(r.cfg, os.Getpid(), token, r.stratumPort)...)
	cmd := exec.Command(r.cfg.ExePath, args...)
	cmd.Dir = filepath.Dir(r.cfg.ExePath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	prepareNodeCommand(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start P2Pool-Go: %w", err)
	}
	if err := afterNodeStart(cmd); err != nil {
		// -parentpid still ends the node if OCM dies.
		logging.Warnf("Unable to tie P2Pool-Go to One-Click Miner's lifetime: %v", err)
	}
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()
	r.cmd = cmd
	r.exited = exited
	r.token = token
	r.startedAt = time.Now()
	r.running = true
	return nil
}

func (r *Runner) supervise(stopCh, supDone chan struct{}) {
	defer close(supDone)
	for {
		r.mu.Lock()
		exited, started := r.exited, r.startedAt
		r.mu.Unlock()

		select {
		case <-stopCh:
			return
		case <-exited:
		}

		r.mu.Lock()
		select {
		case <-stopCh: // died during/after Stop: not a death
			r.mu.Unlock()
			return
		default:
		}
		r.running = false
		now := time.Now()
		if now.Sub(started) >= backoffResetUp {
			r.attempt = 0
		}
		logging.Warnf("P2Pool-Go exited unexpectedly")
		if r.deaths.Record(now) {
			logging.Errorf("P2Pool-Go rapid-failed; not restarting")
			r.rapidFailed = true
			r.supervising = false
			r.mu.Unlock()
			return
		}

		for {
			delay := r.backoff(r.attempt)
			r.attempt++
			r.mu.Unlock()
			select {
			case <-stopCh:
				return
			case <-time.After(delay):
			}
			r.mu.Lock()
			select {
			case <-stopCh:
				r.mu.Unlock()
				return
			default:
			}
			err := r.launchLocked()
			if err == nil {
				break
			}
			logging.Warnf("Unable to restart P2Pool-Go: %v", err)
			if r.deaths.Record(time.Now()) {
				r.rapidFailed = true
				r.supervising = false
				r.mu.Unlock()
				return
			}
		}
		r.mu.Unlock()
	}
}

// Stop ends supervision and stops the node: POST /api/shutdown with the
// launch token and wait for it to exit, at most 10s from the start of Stop,
// then kill it. If the POST is refused outright it kills immediately.
func (r *Runner) Stop() error {
	begin := time.Now()
	r.mu.Lock()
	if !r.supervising && !r.running {
		r.mu.Unlock()
		return nil
	}
	stopCh, supDone := r.stopCh, r.supDone
	if r.supervising {
		close(stopCh)
		r.supervising = false
	}
	cmd, exited, token, running := r.cmd, r.exited, r.token, r.running
	port, budget := r.stratumPort, r.stopWait
	r.mu.Unlock()

	var stopErr error
	if running && cmd != nil {
		deadline := begin.Add(budget)
		err := requestShutdown(port, token, budget)
		if err != nil && isConnRefused(err) {
			// The node never got the request; waiting cannot help.
			deadline = time.Now()
		} else if err != nil {
			logging.Warnf("P2Pool-Go shutdown request failed: %v", err)
		}
		select {
		case <-exited:
		case <-time.After(time.Until(deadline)):
			stopErr = killAndWait(cmd, exited)
		}
	}
	if supDone != nil {
		<-supDone
	}
	if r.stopHook != nil {
		r.stopHook()
	}

	r.mu.Lock()
	if r.cmd == cmd { // not a process a concurrent Start launched
		r.running = false
		r.cmd = nil
	}
	r.mu.Unlock()
	return stopErr
}

// killAndWait kills the node and waits a bounded time for it to exit.
func killAndWait(cmd *exec.Cmd, exited <-chan struct{}) error {
	killErr := cmd.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	select {
	case <-exited:
		return nil
	case <-time.After(killWait):
	}
	if killErr != nil {
		return fmt.Errorf("unable to stop P2Pool-Go: %w", killErr)
	}
	return errors.New("P2Pool-Go did not exit after being killed")
}

// requestShutdown POSTs /api/shutdown with the launch token.
func requestShutdown(port int, token string, timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/shutdown", port)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Shutdown-Token", token)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("shutdown request returned %s", resp.Status)
	}
	return nil
}

// Running reports whether the node process is currently alive.
func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// RapidFailed reports whether supervision gave up after repeated deaths.
func (r *Runner) RapidFailed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rapidFailed
}

// LogTail returns the last n lines of <DataDir>/p2pool.log.
func (r *Runner) LogTail(n int) []string {
	if n <= 0 {
		return nil
	}
	f, err := os.Open(filepath.Join(r.cfg.DataDir, "p2pool.log"))
	if err != nil {
		return nil
	}
	defer f.Close()
	partial := false
	if fi, err := f.Stat(); err == nil && fi.Size() > logTailMaxBytes {
		if _, err := f.Seek(fi.Size()-logTailMaxBytes, io.SeekStart); err == nil {
			partial = true
		}
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	if partial {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:] // drop the cut-off first line
		}
	}
	b = bytes.TrimRight(b, "\r\n")
	if len(b) == 0 {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
