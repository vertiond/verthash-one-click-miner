package p2poolgo

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestBuildArgsLight(t *testing.T) {
	cfg := Config{
		DataDir:        "D",
		VerthashPath:   "V",
		Address:        "A",
		Detection:      Detection{Mode: ModeLight, RPCPort: 5888},
		TrustedPeers:   []string{"a", "b"},
		BootstrapPeers: []string{"x", "y"},
	}
	want := []string{
		"-datadir", "D", "-verthash", "V", "-bind", "127.0.0.1", "-stratum", "9172",
		"-parentpid", "42", "-shutdown-token", "T", "-pooladdr", "A",
		"-trustedpeers", "a,b", "-peers", "x,y",
		"-light", "-p2pbind", "127.0.0.1", "-nonatportmap",
	}
	got := BuildArgs(cfg, 42, "T")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestBuildArgsFull(t *testing.T) {
	base := []string{
		"-datadir", "D", "-verthash", "V", "-bind", "127.0.0.1", "-stratum", "9172",
		"-parentpid", "42", "-shutdown-token", "T", "-pooladdr", "A",
		"-trustedpeers", "a", "-peers", "x",
	}
	cfg := Config{
		DataDir:        "D",
		VerthashPath:   "V",
		Address:        "A",
		Detection:      Detection{Mode: ModeFull, CookiePath: "C", RPCPort: 5888},
		TrustedPeers:   []string{"a"},
		BootstrapPeers: []string{"x"},
	}
	want := append(append([]string{}, base...), "-cookie", "C")
	if got := BuildArgs(cfg, 42, "T"); !reflect.DeepEqual(got, want) {
		t.Fatalf("default rpc port:\ngot  %q\nwant %q", got, want)
	}

	cfg.Detection.RPCPort = 15888
	want = append(append([]string{}, base...), "-cookie", "C", "-rpcport", "15888")
	if got := BuildArgs(cfg, 42, "T"); !reflect.DeepEqual(got, want) {
		t.Fatalf("custom rpc port:\ngot  %q\nwant %q", got, want)
	}
}

func TestBuildArgsFullWithConfCredentials(t *testing.T) {
	cfg := Config{
		DataDir:      "D",
		VerthashPath: "V",
		Address:      "A",
		Detection:    Detection{Mode: ModeFull, RPCUser: "vert", RPCPass: "pw", RPCPort: 15888},
	}
	want := []string{
		"-datadir", "D", "-verthash", "V", "-bind", "127.0.0.1", "-stratum", "9172",
		"-parentpid", "42", "-shutdown-token", "T", "-pooladdr", "A",
		"-u", "vert", "-p", "pw", "-rpcport", "15888",
	}
	if got := BuildArgs(cfg, 42, "T"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestBuildArgsOmitsEmptyPeerLists(t *testing.T) {
	cfg := Config{
		DataDir:      "D",
		VerthashPath: "V",
		Address:      "A",
		Detection:    Detection{Mode: ModeLight},
	}
	want := []string{
		"-datadir", "D", "-verthash", "V", "-bind", "127.0.0.1", "-stratum", "9172",
		"-parentpid", "42", "-shutdown-token", "T", "-pooladdr", "A",
		"-light", "-p2pbind", "127.0.0.1", "-nonatportmap",
	}
	if got := BuildArgs(cfg, 42, "T"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestNextBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		0:  2 * time.Second,
		1:  4 * time.Second,
		2:  8 * time.Second,
		5:  60 * time.Second,
		10: 60 * time.Second,
		40: 60 * time.Second,
	}
	for attempt, want := range cases {
		if got := NextBackoff(attempt); got != want {
			t.Errorf("NextBackoff(%d) = %v, want %v", attempt, got, want)
		}
	}
}

func TestDeathLogRapidFail(t *testing.T) {
	start := time.Now()
	var d deathLog
	for i := 0; i < 4; i++ {
		if d.Record(start.Add(time.Duration(i) * 15 * time.Second)) {
			t.Fatalf("death %d reported rapid-fail", i+1)
		}
	}
	if !d.Record(start.Add(60 * time.Second)) {
		t.Fatal("5th death within 2 min did not report rapid-fail")
	}

	var spread deathLog
	for i := 0; i < 5; i++ {
		if spread.Record(start.Add(time.Duration(i) * 45 * time.Second)) {
			t.Fatalf("death %d of 5 spread over 3 min reported rapid-fail", i+1)
		}
	}
}

func TestPortFree(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if PortFree(addr) {
		t.Fatalf("%s reported free while listening", addr)
	}
	l.Close()
	if !PortFree(addr) {
		t.Fatalf("%s reported busy after close", addr)
	}
}

func TestRunnerStartFailsOnBusyPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	busy := l.Addr().(*net.TCPAddr).Port

	r := NewRunner(Config{ExePath: os.Args[0], DataDir: t.TempDir(), Detection: Detection{Mode: ModeLight}})
	r.stratumPort = busy
	r.p2pPort = freePort(t)
	err = r.Start()
	want := fmt.Sprintf("Port %d is already in use by another program. Close it and retry.", busy)
	if err == nil || err.Error() != want {
		t.Fatalf("Start() error = %v, want %q", err, want)
	}
	if r.Running() {
		t.Fatal("runner reports running after a port conflict")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

const helperEnv = "OCM_P2POOLGO_HELPER_NODE"
const helperMarkerEnv = "OCM_P2POOLGO_HELPER_MARKER"

func TestRunnerRestartsThenRapidFails(t *testing.T) {
	t.Setenv(helperEnv, "crash")
	r := NewRunner(Config{ExePath: os.Args[0], DataDir: t.TempDir(), Detection: Detection{Mode: ModeLight}})
	r.stratumPort = freePort(t)
	r.p2pPort = freePort(t)
	r.argPrefix = []string{"-test.run=^TestHelperNode$", "--"}
	var attempts []int
	r.backoff = func(attempt int) time.Duration {
		attempts = append(attempts, attempt) // called with r.mu held
		return 10 * time.Millisecond
	}
	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer r.Stop()
	deadline := time.Now().Add(30 * time.Second)
	for !r.RapidFailed() {
		if time.Now().After(deadline) {
			t.Fatal("crashing node never rapid-failed")
		}
		time.Sleep(20 * time.Millisecond)
	}
	r.mu.Lock()
	deaths, got := len(r.deaths.times), append([]int{}, attempts...)
	r.mu.Unlock()
	if deaths != 5 {
		t.Fatalf("deaths = %d, want 5", deaths)
	}
	if want := []int{0, 1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("backoff attempts = %v, want %v", got, want)
	}
	if r.Running() {
		t.Fatal("Running() true after rapid-fail")
	}
}

func TestRunnerGracefulStop(t *testing.T) {
	marker := t.TempDir() + string(os.PathSeparator) + "shutdown.txt"
	t.Setenv(helperEnv, "1")
	t.Setenv(helperMarkerEnv, marker)

	dataDir := t.TempDir()
	r := NewRunner(Config{
		ExePath:      os.Args[0],
		DataDir:      dataDir,
		VerthashPath: "V",
		Address:      "A",
		Detection:    Detection{Mode: ModeLight},
	})
	r.stratumPort = freePort(t)
	r.p2pPort = freePort(t)
	r.argPrefix = []string{"-test.run=^TestHelperNode$", "--"}

	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !r.Running() {
		t.Fatal("Running() false right after Start")
	}
	r.mu.Lock()
	token := r.token
	r.mu.Unlock()
	if len(token) != 32 {
		t.Fatalf("token %q is not 16 hex-encoded bytes", token)
	}

	// Wait for the helper to listen, so Stop takes the graceful path.
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(r.stratumPort))
	deadline := time.Now().Add(20 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			r.Stop()
			t.Fatalf("helper never listened on %s", addr)
		}
		time.Sleep(50 * time.Millisecond)
	}

	begin := time.Now()
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if took := time.Since(begin); took > 10*time.Second {
		t.Fatalf("Stop took %v, want ≤10s", took)
	}
	if r.Running() {
		t.Fatal("Running() true after Stop")
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("helper never received a valid shutdown: %v", err)
	}
	if string(got) != token {
		t.Fatalf("helper saw token %q, want %q", got, token)
	}

	// A requested stop must not count as a death or trigger a restart.
	time.Sleep(2500 * time.Millisecond)
	if r.Running() || r.RapidFailed() {
		t.Fatal("runner restarted or rapid-failed after a requested stop")
	}
	r.mu.Lock()
	deaths := len(r.deaths.times)
	r.mu.Unlock()
	if deaths != 0 {
		t.Fatalf("requested stop recorded %d deaths", deaths)
	}
}

// startHelperRunner starts a runner whose node is the TestHelperNode helper
// in the given mode, on free ephemeral ports.
func startHelperRunner(t *testing.T, mode string) *Runner {
	t.Helper()
	t.Setenv(helperEnv, mode)
	t.Setenv(helperMarkerEnv, filepath.Join(t.TempDir(), "shutdown.txt"))
	r := NewRunner(Config{ExePath: os.Args[0], DataDir: t.TempDir(), Detection: Detection{Mode: ModeLight}})
	r.stratumPort = freePort(t)
	r.p2pPort = freePort(t)
	r.argPrefix = []string{"-test.run=^TestHelperNode$", "--"}
	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return r
}

func waitListening(t *testing.T, r *Runner) {
	t.Helper()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(r.stratumPort))
	deadline := time.Now().Add(20 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		if time.Now().After(deadline) {
			r.Stop()
			t.Fatalf("helper never listened on %s", addr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func assertStoppedClean(t *testing.T, r *Runner, exited chan struct{}) {
	t.Helper()
	select {
	case <-exited:
	default:
		t.Fatal("node process still alive after Stop")
	}
	time.Sleep(2500 * time.Millisecond)
	if r.Running() || r.RapidFailed() {
		t.Fatal("runner restarted or rapid-failed after a requested stop")
	}
	r.mu.Lock()
	deaths := len(r.deaths.times)
	r.mu.Unlock()
	if deaths != 0 {
		t.Fatalf("requested stop recorded %d deaths", deaths)
	}
}

func TestRunnerStopKillsWhenShutdownIgnored(t *testing.T) {
	r := startHelperRunner(t, "ignore")
	r.mu.Lock()
	r.stopWait = time.Second
	exited := r.exited
	r.mu.Unlock()
	waitListening(t, r)

	begin := time.Now()
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	took := time.Since(begin)
	if took < 900*time.Millisecond || took > 3*time.Second {
		t.Fatalf("Stop took %v, want about the 1s budget", took)
	}
	if _, err := os.Stat(os.Getenv(helperMarkerEnv)); err != nil {
		t.Fatalf("helper never received the shutdown request: %v", err)
	}
	assertStoppedClean(t, r, exited)
}

func TestRunnerStopKillsImmediatelyWhenNotListening(t *testing.T) {
	r := startHelperRunner(t, "silent")
	r.mu.Lock()
	exited := r.exited // default 10s budget
	r.mu.Unlock()

	begin := time.Now()
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Windows reports a refused loopback connect after about 2s.
	if took := time.Since(begin); took > 5*time.Second {
		t.Fatalf("Stop took %v, want well under the 10s budget", took)
	}
	assertStoppedClean(t, r, exited)
}

func TestRunnerStopKeepsConcurrentlyStartedProcess(t *testing.T) {
	r := startHelperRunner(t, "1")
	waitListening(t, r)
	var startErr error
	// A Start that lands after Stop released the lock but before it finished.
	r.stopHook = func() { startErr = r.Start() }

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	r.stopHook = nil
	if startErr != nil {
		t.Fatalf("concurrent Start: %v", startErr)
	}
	if !r.Running() {
		t.Fatal("Stop cleared the state of a process a concurrent Start launched")
	}
	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()
	if cmd == nil {
		t.Fatal("Stop cleared cmd of the concurrently started process")
	}
	waitListening(t, r)
	if err := r.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if r.Running() {
		t.Fatal("Running() true after second Stop")
	}
}

func TestRunnerLogTail(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(Config{DataDir: dir})
	if got := r.LogTail(3); len(got) != 0 {
		t.Fatalf("missing log: got %q", got)
	}
	if err := os.WriteFile(dir+string(os.PathSeparator)+"p2pool.log", []byte("one\ntwo\r\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := r.LogTail(3), []string{"two", "three", "four"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got, want := r.LogTail(10), []string{"one", "two", "three", "four"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestHelperNode is not a real test: TestRunnerGracefulStop re-execs the test
// binary with it as a stand-in p2pool-go node.
func TestHelperNode(t *testing.T) {
	mode := os.Getenv(helperEnv)
	switch mode {
	case "1", "ignore":
	case "crash":
		os.Exit(1)
	case "silent": // never listens
		time.Sleep(30 * time.Second)
		os.Exit(3)
	default:
		t.Skip("helper process only")
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	var port, token string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-stratum":
			port = args[i+1]
		case "-shutdown-token":
			token = args[i+1]
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, req *http.Request) {
		got := req.Header.Get("X-Shutdown-Token")
		if req.Method != http.MethodPost || got != token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		os.WriteFile(os.Getenv(helperMarkerEnv), []byte(got), 0o644)
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if mode == "ignore" {
			return
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	})
	go func() {
		time.Sleep(30 * time.Second)
		os.Exit(3)
	}()
	if err := http.ListenAndServe("127.0.0.1:"+port, mux); err != nil {
		os.Exit(2)
	}
}

func TestP2PCheckAddrIsLoopbackInEveryMode(t *testing.T) {
	port := freePort(t)
	for _, mode := range []Mode{ModeLight, ModeFull} {
		r := NewRunner(Config{Detection: Detection{Mode: mode}})
		r.p2pPort = port
		want := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		if got := r.p2pCheckAddr(); got != want {
			t.Fatalf("mode %d: p2pCheckAddr() = %q, want %q", mode, got, want)
		}
	}
}

func TestPortAnswers(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	if !portAnswers(addr) {
		t.Fatalf("%s did not answer while listening", addr)
	}
	l.Close()
	if portAnswers(addr) {
		t.Fatalf("%s answered after close", addr)
	}
}
