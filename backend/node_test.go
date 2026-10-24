package backend

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/p2poolgo"
	"github.com/vertcoin-project/one-click-miner-vnext/tracking"
)

// fakeNode is a nodeController that records calls.
type fakeNode struct {
	mu          sync.Mutex
	cfg         p2poolgo.Config
	running     bool
	rapidFailed bool
	startErr    error
	stopErr     error
	starts      int
	stops       int
	tail        []string
}

func (f *fakeNode) Config() p2poolgo.Config { return f.cfg }

func (f *fakeNode) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	if f.startErr != nil {
		return f.startErr
	}
	f.running = true
	return nil
}

func (f *fakeNode) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	if f.stopErr != nil {
		return f.stopErr
	}
	f.running = false
	return nil
}

func (f *fakeNode) Running() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

func (f *fakeNode) RapidFailed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rapidFailed
}

func (f *fakeNode) LogTail(n int) []string { return f.tail }

func (f *fakeNode) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.starts, f.stops
}

type emitted struct {
	mu     sync.Mutex
	events []string // "name:arg0"
}

func (e *emitted) emit(name string, data ...interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := name
	if len(data) > 0 {
		s += ":" + fmtArg(data[0])
	}
	e.events = append(e.events, s)
}

func fmtArg(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return "?"
}

func (e *emitted) list() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.events...)
}

// stubNodeEnv replaces every injectable with a test double and restores
// them when the test ends. The returned counters track calls.
type nodeEnv struct {
	runnersBuilt []*fakeNode
	installs     int
	detects      int
	existing     bool
}

func stubNodeEnv(t *testing.T) *nodeEnv {
	t.Helper()
	env := &nodeEnv{}
	oldNewRunner, oldDetectExisting, oldLoad, oldCore := newRunner, detectExisting, loadAndInstall, detectCore
	oldStatus, oldPoll, oldSlow, oldAddr, oldDir := getStatus, nodeReadyPoll, nodeSlowAfter, nodeAddress, nodeDataDir
	oldCoreRunning := coreRunning
	t.Cleanup(func() {
		newRunner, detectExisting, loadAndInstall, detectCore = oldNewRunner, oldDetectExisting, oldLoad, oldCore
		getStatus, nodeReadyPoll, nodeSlowAfter, nodeAddress, nodeDataDir = oldStatus, oldPoll, oldSlow, oldAddr, oldDir
		coreRunning = oldCoreRunning
	})
	coreRunning = func() bool { return true }
	dir := t.TempDir()
	newRunner = func(cfg p2poolgo.Config) nodeController {
		f := &fakeNode{cfg: cfg}
		env.runnersBuilt = append(env.runnersBuilt, f)
		return f
	}
	detectExisting = func(string, time.Duration) bool { return env.existing }
	loadAndInstall = func() (string, []string, []string, error) {
		env.installs++
		return "/bin/p2pool", []string{"trusted:9348"}, []string{"boot:9348"}, nil
	}
	detectCore = func() p2poolgo.Detection {
		env.detects++
		return p2poolgo.Detection{Mode: p2poolgo.ModeLight, RPCPort: 5888}
	}
	getStatus = func(string, string) (*p2poolgo.Status, error) { return nil, errors.New("connection refused") }
	nodeReadyPoll = time.Millisecond
	nodeSlowAfter = time.Hour
	nodeAddress = func(*Backend) string { return "Vaddr1" }
	nodeDataDir = func() string { return dir }
	return env
}

func testBackend(e *emitted) *Backend {
	return &Backend{emit: e.emit, nodeRetry: make(chan bool, 1)}
}

func TestPrepareNodeReusesHealthyNode(t *testing.T) {
	env := stubNodeEnv(t)
	existingCalled := false
	detectExisting = func(string, time.Duration) bool { existingCalled = true; return false }
	m := testBackend(&emitted{})
	old := &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, running: true}
	m.node = old

	if err := m.PrepareNode(); err != nil {
		t.Fatal(err)
	}
	if err := m.StartNode(); err != nil {
		t.Fatal(err)
	}
	starts, stops := old.counts()
	if starts != 0 || stops != 0 {
		t.Errorf("starts=%d stops=%d, want 0/0", starts, stops)
	}
	if env.installs != 0 || env.detects != 0 || existingCalled || len(env.runnersBuilt) != 0 {
		t.Errorf("installs=%d detects=%d existing=%v runners=%d, want none", env.installs, env.detects, existingCalled, len(env.runnersBuilt))
	}
	if m.node != old {
		t.Error("the healthy node was replaced")
	}
}

func TestPrepareNodeRestartsOnAddressChange(t *testing.T) {
	env := stubNodeEnv(t)
	m := testBackend(&emitted{})
	old := &fakeNode{cfg: p2poolgo.Config{Address: "Vold"}, running: true}
	m.node = old

	if err := m.PrepareNode(); err != nil {
		t.Fatal(err)
	}
	if _, stops := old.counts(); stops != 1 {
		t.Errorf("old node stops=%d, want 1", stops)
	}
	if len(env.runnersBuilt) != 1 {
		t.Fatalf("runners built=%d, want 1", len(env.runnersBuilt))
	}
	nr := env.runnersBuilt[0]
	cfg := nr.Config()
	if cfg.Address != "Vaddr1" || cfg.ExePath != "/bin/p2pool" || cfg.DataDir != nodeDataDir() {
		t.Errorf("unexpected config %+v", cfg)
	}
	if len(cfg.TrustedPeers) != 1 || len(cfg.BootstrapPeers) != 1 {
		t.Errorf("peers not passed: %+v", cfg)
	}
	if starts, _ := nr.counts(); starts != 0 {
		t.Errorf("PrepareNode started the node")
	}
	if err := m.StartNode(); err != nil {
		t.Fatal(err)
	}
	if starts, _ := nr.counts(); starts != 1 {
		t.Errorf("new node starts=%d, want 1", starts)
	}
}

func TestPrepareNodeAdoptsExisting(t *testing.T) {
	env := stubNodeEnv(t)
	env.existing = true
	m := testBackend(&emitted{})

	if err := m.PrepareNode(); err != nil {
		t.Fatal(err)
	}
	if !m.adopted {
		t.Error("adopted not set")
	}
	if len(env.runnersBuilt) != 0 || env.installs != 0 || env.detects != 0 {
		t.Errorf("runners=%d installs=%d detects=%d, want none", len(env.runnersBuilt), env.installs, env.detects)
	}
	if err := m.StartNode(); err != nil {
		t.Fatalf("StartNode when adopted: %v", err)
	}
	if n := m.GetRPCNotice(); n.Show {
		t.Error("RPC notice shown for an adopted node")
	}
}

func TestStartNodeReturnsPortConflict(t *testing.T) {
	env := stubNodeEnv(t)
	newRunner = func(cfg p2poolgo.Config) nodeController {
		f := &fakeNode{cfg: cfg, startErr: errors.New("Port 9172 is already in use by another program. Close it and retry.")}
		env.runnersBuilt = append(env.runnersBuilt, f)
		return f
	}
	m := testBackend(&emitted{})
	if err := m.PrepareNode(); err != nil {
		t.Fatal(err)
	}
	err := m.StartNode()
	if err == nil || err.Error() != "Port 9172 is already in use by another program. Close it and retry." {
		t.Fatalf("StartNode error = %v", err)
	}
}

func TestPrepareNodeNoBinaryForPlatform(t *testing.T) {
	stubNodeEnv(t)
	loadAndInstall = func() (string, []string, []string, error) {
		return "", nil, nil, errNoNodeBinary
	}
	m := testBackend(&emitted{})
	err := m.PrepareNode()
	if err == nil || err.Error() != "P2Pool-Go is not available for this platform." {
		t.Fatalf("PrepareNode error = %v", err)
	}
}

func nodeStates(events []string) []string {
	var out []string
	for _, e := range events {
		if strings.HasPrefix(e, "checkStatus:node_") {
			out = append(out, strings.TrimPrefix(e, "checkStatus:"))
		}
	}
	return out
}

func TestWaitNodeReadyEmitsStates(t *testing.T) {
	stubNodeEnv(t)
	seq := []*p2poolgo.Status{
		{State: "waiting_template", Version: "v"},
		{State: "syncing_sharechain", Version: "v", Peers: 3},
		{State: "ready", Version: "v", Ready: true},
	}
	i := 0
	getStatus = func(string, string) (*p2poolgo.Status, error) {
		s := seq[i]
		if i < len(seq)-1 {
			i++
		}
		return s, nil
	}
	e := &emitted{}
	m := testBackend(e)
	m.node = &fakeNode{running: true}

	if err := m.WaitNodeReady(); err != nil {
		t.Fatal(err)
	}
	got := nodeStates(e.list())
	want := []string{"node_waiting_template", "node_syncing_sharechain", "node_ready"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("states = %v, want %v", got, want)
	}
}

func TestWaitNodeReadyConnectionErrorIsStarting(t *testing.T) {
	stubNodeEnv(t)
	calls := 0
	getStatus = func(string, string) (*p2poolgo.Status, error) {
		calls++
		if calls < 3 {
			return nil, errors.New("connection refused")
		}
		return &p2poolgo.Status{State: "ready", Version: "v", Ready: true}, nil
	}
	e := &emitted{}
	m := testBackend(e)
	m.node = &fakeNode{running: true}
	if err := m.WaitNodeReady(); err != nil {
		t.Fatal(err)
	}
	got := nodeStates(e.list())
	if len(got) < 2 || got[0] != "node_starting" || got[1] != "node_starting" {
		t.Errorf("states = %v, want node_starting first", got)
	}
}

func TestWaitNodeReadyRapidFailReturnsLog(t *testing.T) {
	stubNodeEnv(t)
	m := testBackend(&emitted{})
	m.node = &fakeNode{rapidFailed: true, tail: []string{"line one", "fatal: bad thing"}}

	done := make(chan error, 1)
	go func() { done <- m.WaitNodeReady() }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "line one\nfatal: bad thing") {
			t.Fatalf("error = %v, want the log tail", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitNodeReady did not return on rapid-fail")
	}
}

func TestWaitNodeReadySlowAndRetry(t *testing.T) {
	stubNodeEnv(t)
	nodeSlowAfter = 20 * time.Millisecond
	var mu sync.Mutex
	ready := false
	getStatus = func(string, string) (*p2poolgo.Status, error) {
		mu.Lock()
		defer mu.Unlock()
		if ready {
			return &p2poolgo.Status{State: "ready", Version: "v", Ready: true}, nil
		}
		return &p2poolgo.Status{State: "syncing_sharechain", Version: "v"}, nil
	}
	e := &emitted{}
	m := testBackend(e)
	node := &fakeNode{running: true}
	m.node = node

	done := make(chan error, 1)
	go func() { done <- m.WaitNodeReady() }()

	waitFor(t, func() bool { return contains(e.list(), "nodeSlow:true") })
	m.RetryNode()
	waitFor(t, func() bool { return contains(e.list(), "nodeSlow:false") })
	starts, stops := node.counts()
	if starts != 1 || stops != 1 {
		t.Errorf("retry: starts=%d stops=%d, want 1/1", starts, stops)
	}
	mu.Lock()
	ready = true
	mu.Unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	slowTrue := 0
	for _, ev := range e.list() {
		if ev == "nodeSlow:true" {
			slowTrue++
		}
	}
	if slowTrue < 1 {
		t.Error("nodeSlow true never emitted")
	}
}

func TestShutdownNodeStopsOwnedAndEndsWait(t *testing.T) {
	stubNodeEnv(t)
	getStatus = func(string, string) (*p2poolgo.Status, error) {
		return &p2poolgo.Status{State: "syncing_sharechain", Version: "v"}, nil
	}
	m := testBackend(&emitted{})
	node := &fakeNode{running: true}
	m.node = node
	done := make(chan error, 1)
	go func() { done <- m.WaitNodeReady() }()
	time.Sleep(10 * time.Millisecond)
	m.ShutdownNode()
	if _, stops := node.counts(); stops != 1 {
		t.Errorf("stops=%d, want 1", stops)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Error("WaitNodeReady returned nil after shutdown")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitNodeReady kept waiting after shutdown")
	}
	if err := m.StartNode(); err == nil {
		t.Error("StartNode after shutdown succeeded")
	}
}

func TestShutdownNodeAdoptedIsNoop(t *testing.T) {
	stubNodeEnv(t)
	m := testBackend(&emitted{})
	m.adopted = true
	m.ShutdownNode() // must not panic with no node
	var nilBackend Backend
	nilBackend.ShutdownNode()
}

func TestCheckVerthashWithRunningNode(t *testing.T) {
	env := stubNodeEnv(t)
	oldVerify, oldEnsure := verifyVerthash, ensureVerthash
	t.Cleanup(func() { verifyVerthash, ensureVerthash = oldVerify, oldEnsure })

	ensured := 0
	ensureVerthash = func(string, bool, chan float64) error { ensured++; return nil }

	// Valid file: verify only, node untouched.
	verifyVerthash = func(string) (bool, error) { return true, nil }
	m := testBackend(&emitted{})
	node := &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, running: true}
	m.node = node
	if err := m.checkVerthash("v.dat", false); err != nil {
		t.Fatal(err)
	}
	if _, stops := node.counts(); stops != 0 || ensured != 0 {
		t.Errorf("valid: stops=%d ensured=%d, want 0/0", stops, ensured)
	}

	// Invalid file: stop, ensure, and StartNode starts a fresh runner.
	verifyVerthash = func(string) (bool, error) { return false, nil }
	if err := m.checkVerthash("v.dat", false); err != nil {
		t.Fatal(err)
	}
	if _, stops := node.counts(); stops != 1 || ensured != 1 {
		t.Errorf("invalid: stops=%d ensured=%d, want 1/1", stops, ensured)
	}
	if err := m.StartNode(); err != nil {
		t.Fatal(err)
	}
	if len(env.runnersBuilt) != 1 {
		t.Fatalf("fresh runners=%d, want 1", len(env.runnersBuilt))
	}
	if starts, _ := env.runnersBuilt[0].counts(); starts != 1 {
		t.Errorf("fresh runner starts=%d, want 1", starts)
	}
	if env.runnersBuilt[0].Config().Address != "Vaddr1" {
		t.Error("fresh runner lost the config")
	}
}

func TestCheckVerthashWithoutNodeEnsures(t *testing.T) {
	stubNodeEnv(t)
	oldVerify, oldEnsure := verifyVerthash, ensureVerthash
	t.Cleanup(func() { verifyVerthash, ensureVerthash = oldVerify, oldEnsure })
	verified, ensured := 0, 0
	verifyVerthash = func(string) (bool, error) { verified++; return true, nil }
	ensureVerthash = func(string, bool, chan float64) error { ensured++; return nil }
	m := testBackend(&emitted{})
	if err := m.checkVerthash("v.dat", false); err != nil {
		t.Fatal(err)
	}
	if verified != 0 || ensured != 1 {
		t.Errorf("verified=%d ensured=%d, want 0/1", verified, ensured)
	}
}

func TestGetRPCNotice(t *testing.T) {
	stubNodeEnv(t)
	m := testBackend(&emitted{})
	m.detection = p2poolgo.Detection{RPCNotice: true, ConfPath: `C:\Users\x\AppData\Roaming\Vertcoin\vertcoin.conf`}
	n := m.GetRPCNotice()
	if !n.Show || n.ConfPath != m.detection.ConfPath || n.Sample != "server=1" {
		t.Errorf("notice = %+v", n)
	}
	m.detection = p2poolgo.Detection{}
	if m.GetRPCNotice().Show {
		t.Error("notice shown without RPCNotice")
	}
}

func TestGetRPCNoticeHiddenOnceCoreStops(t *testing.T) {
	stubNodeEnv(t)
	m := testBackend(&emitted{})
	m.detection = p2poolgo.Detection{RPCNotice: true, ConfPath: "vertcoin.conf"}
	running := true
	coreRunning = func() bool { return running }
	if !m.GetRPCNotice().Show {
		t.Fatal("notice hidden while Core runs")
	}
	running = false
	if n := m.GetRPCNotice(); n.Show {
		t.Errorf("notice shown after Core stopped: %+v", n)
	}
	if !m.detection.RPCNotice || m.detection.Mode != p2poolgo.ModeLight {
		t.Errorf("detection changed: %+v", m.detection)
	}
}

func TestNetworkNumbers(t *testing.T) {
	stubNodeEnv(t)
	oldDiff, oldTip, oldHasBackend := backendDifficulty, backendTipHeight, hasBackend
	t.Cleanup(func() { backendDifficulty, backendTipHeight, hasBackend = oldDiff, oldTip, oldHasBackend })

	backendCalls, statusCalls := 0, 0
	backendDifficulty = func() float64 { backendCalls++; return 100 }
	backendTipHeight = func() int64 { return 1000 }
	have := false
	hasBackend = func() bool { return have }
	node := &p2poolgo.Status{Ready: true, State: "ready", Version: "v", Difficulty: 120, Height: 1001}
	getStatus = func(string, string) (*p2poolgo.Status, error) { statusCalls++; return node, nil }

	m := testBackend(&emitted{})
	var ns networkState

	// Cycle 0 without a backend: node numbers accepted, backend retried later.
	diff, tip := m.networkNumbers(0, &ns)
	if diff != 120 || tip != 1001 || backendCalls != 0 || statusCalls != 1 {
		t.Fatalf("cycle0: diff=%v tip=%v backend=%d status=%d", diff, tip, backendCalls, statusCalls)
	}
	// Backend appears: refreshed on the next cycle even though not a 600 boundary.
	have = true
	m.networkNumbers(1, &ns)
	if backendCalls != 1 || m.lastBackendDiff != 100 {
		t.Fatalf("backend not refreshed once available: calls=%d last=%v", backendCalls, m.lastBackendDiff)
	}
	// No further polls until the boundaries.
	m.networkNumbers(2, &ns)
	if backendCalls != 1 || statusCalls != 1 {
		t.Errorf("polled too often: backend=%d status=%d", backendCalls, statusCalls)
	}
	m.networkNumbers(30, &ns)
	if statusCalls != 2 {
		t.Errorf("status not polled at 30: %d", statusCalls)
	}
	m.networkNumbers(600, &ns)
	if backendCalls != 2 {
		t.Errorf("backend not refreshed at 600: %d", backendCalls)
	}
	// Node difficulty out of range: backend numbers used.
	node = &p2poolgo.Status{Ready: true, State: "ready", Version: "v", Difficulty: 100000, Height: 1001}
	diff, tip = m.networkNumbers(630, &ns)
	if diff != 100 || tip != 1000 {
		t.Errorf("out-of-range: diff=%v tip=%v, want backend 100/1000", diff, tip)
	}
	// Node unreachable: backend numbers.
	getStatus = func(string, string) (*p2poolgo.Status, error) { return nil, errors.New("down") }
	diff, tip = m.networkNumbers(660, &ns)
	if diff != 100 || tip != 1000 {
		t.Errorf("unreachable: diff=%v tip=%v", diff, tip)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestNodeRapidFailDuringMiningEmitsEvent(t *testing.T) {
	stubNodeEnv(t)
	e := &emitted{}
	m := testBackend(e)
	node := &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, running: true}
	m.node = node
	reported := false

	m.watchNodeRapidFail(&reported)
	if contains(e.list(), "nodeRapidFail") {
		t.Fatal("event emitted for a healthy node")
	}
	node.mu.Lock()
	node.rapidFailed, node.running = true, false
	node.mu.Unlock()
	for i := 0; i < 3; i++ {
		m.watchNodeRapidFail(&reported)
	}
	n := 0
	for _, ev := range e.list() {
		if ev == "nodeRapidFail" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("nodeRapidFail emitted %d times, want 1", n)
	}

	// An adopted node is not ours to report.
	e2 := &emitted{}
	m2 := testBackend(e2)
	m2.adopted = true
	m2.node = &fakeNode{rapidFailed: true}
	r2 := false
	m2.watchNodeRapidFail(&r2)
	if contains(e2.list(), "nodeRapidFail") {
		t.Error("event emitted for an adopted node")
	}
}

func TestPrepareNodeReportsRapidFailedNodeLogTail(t *testing.T) {
	env := stubNodeEnv(t)
	m := testBackend(&emitted{})
	m.node = &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, rapidFailed: true,
		tail: []string{"sharechain corrupt", "exiting"}}

	err := m.PrepareNode()
	if err == nil || !strings.Contains(err.Error(), "sharechain corrupt\nexiting") {
		t.Fatalf("PrepareNode error = %v, want the log tail", err)
	}
	if env.installs != 0 || len(env.runnersBuilt) != 0 {
		t.Errorf("installs=%d runners=%d, want none on the failing run", env.installs, len(env.runnersBuilt))
	}
	// The next checks run (the user's retry) builds a fresh runner.
	if err := m.PrepareNode(); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(env.runnersBuilt) != 1 {
		t.Errorf("retry built %d runners, want 1", len(env.runnersBuilt))
	}
}

func TestCheckVerthashVerifiesWhenOwnedNodeInBackoff(t *testing.T) {
	env := stubNodeEnv(t)
	oldVerify, oldEnsure := verifyVerthash, ensureVerthash
	t.Cleanup(func() { verifyVerthash, ensureVerthash = oldVerify, oldEnsure })
	verified, ensured := 0, 0
	verifyVerthash = func(string) (bool, error) { verified++; return true, nil }
	ensureVerthash = func(string, bool, chan float64) error { ensured++; return nil }

	m := testBackend(&emitted{})
	node := &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, running: false} // between relaunches
	m.node = node
	if err := m.checkVerthash("v.dat", false); err != nil {
		t.Fatal(err)
	}
	if verified != 1 || ensured != 0 {
		t.Errorf("verified=%d ensured=%d, want 1/0", verified, ensured)
	}
	if _, stops := node.counts(); stops != 0 || len(env.runnersBuilt) != 0 {
		t.Errorf("stops=%d runners=%d, want 0/0", stops, len(env.runnersBuilt))
	}
}

func TestCheckVerthashStopErrorAbortsRegenerate(t *testing.T) {
	stubNodeEnv(t)
	oldVerify, oldEnsure := verifyVerthash, ensureVerthash
	t.Cleanup(func() { verifyVerthash, ensureVerthash = oldVerify, oldEnsure })
	ensured := 0
	verifyVerthash = func(string) (bool, error) { return false, nil }
	ensureVerthash = func(string, bool, chan float64) error { ensured++; return nil }

	m := testBackend(&emitted{})
	m.node = &fakeNode{running: true, stopErr: errors.New("access denied")}
	err := m.checkVerthash("v.dat", false)
	if err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("checkVerthash error = %v, want the stop error", err)
	}
	if ensured != 0 {
		t.Error("verthash regenerated after a failed stop")
	}
}

func TestPrepareNodeStopErrorAbortsInstall(t *testing.T) {
	env := stubNodeEnv(t)
	m := testBackend(&emitted{})
	old := &fakeNode{cfg: p2poolgo.Config{Address: "Vold"}, running: true, stopErr: errors.New("access denied")}
	m.node = old

	err := m.PrepareNode()
	if err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("PrepareNode error = %v, want the stop error", err)
	}
	if env.installs != 0 || len(env.runnersBuilt) != 0 {
		t.Errorf("installs=%d runners=%d, want none", env.installs, len(env.runnersBuilt))
	}
	if m.node != old {
		t.Error("the node that failed to stop was dropped; quit could not stop it")
	}
}

func TestPrepareNodeRapidFailTracksNoLogContent(t *testing.T) {
	stubNodeEnv(t)
	oldTrack := track
	t.Cleanup(func() { track = oldTrack })
	var got []tracking.TrackingRequest
	track = func(r tracking.TrackingRequest) { got = append(got, r) }

	m := testBackend(&emitted{})
	m.node = &fakeNode{cfg: p2poolgo.Config{Address: "Vaddr1"}, rapidFailed: true,
		tail: []string{"payout to Vaddr1 via peer 203.0.113.7:9348", `C:\Users\someone\AppData\p2pool-go`}}
	err := m.PrepareNode()
	if err == nil {
		t.Fatal("PrepareNode succeeded for a rapid-failed node")
	}
	trackNodeFailure("PrepareNodeError", err)

	if len(got) != 1 {
		t.Fatalf("tracked %d requests, want 1", len(got))
	}
	r := got[0]
	if r.Name != "RapidFail" {
		t.Errorf("Name = %q, want RapidFail", r.Name)
	}
	for _, leak := range []string{"Vaddr1", "203.0.113.7", "someone", "p2pool.log"} {
		if strings.Contains(r.Name+r.Action+r.Category, leak) {
			t.Errorf("tracking request %+v leaks %q", r, leak)
		}
	}
}

func TestNodeFailureKind(t *testing.T) {
	cases := map[string]error{
		"RapidFail":           &nodeRapidFailError{msg: "x"},
		"NoBinary":            errNoNodeBinary,
		"Shutdown":            errNodeShutdown,
		"PortInUse":           errors.New("Port 9172 is already in use by another program. Close it and retry."),
		"ManifestUnavailable": errors.New("Unable to download the P2Pool-Go manifest. Check your internet connection."),
		"Other":               errors.New(`open C:\Users\someone\secret: access denied`),
	}
	for want, err := range cases {
		if got := nodeFailureKind(err); got != want {
			t.Errorf("nodeFailureKind(%q) = %q, want %q", err, got, want)
		}
	}
}
