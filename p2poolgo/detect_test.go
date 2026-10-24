package p2poolgo

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func rpcServer(t *testing.T, ibd bool, status int) (port int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if status != 200 {
			w.WriteHeader(status)
			return
		}
		if !ok || u != "__cookie__" || p != "abc" || r.Method != "POST" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprintf(w, `{"result":{"initialblockdownload":%v},"error":null,"id":"ocm"}`, ibd)
	}))
	t.Cleanup(srv.Close)
	uu, _ := url.Parse(srv.URL)
	port, _ = strconv.Atoi(uu.Port())
	return port
}

func closedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func writeConf(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "vertcoin.conf"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func writeCookie(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".cookie"), []byte("__cookie__:abc\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func env(dir string, running bool) CoreEnv {
	return CoreEnv{DataDir: dir, ProcessRunning: func() bool { return running }, RPCHost: "127.0.0.1"}
}

func TestDetectNotRunning(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, dir, fmt.Sprintf("rpcport=%d\n", closedPort(t)))
	d := DetectCore(env(dir, false))
	if d.Mode != ModeLight || d.RPCNotice {
		t.Fatalf("got %+v", d)
	}
	if d.ConfPath != filepath.Join(dir, "vertcoin.conf") {
		t.Fatalf("ConfPath %q", d.ConfPath)
	}
}

func TestDetectSyncedCoreFullMode(t *testing.T) {
	dir := t.TempDir()
	port := rpcServer(t, false, 200)
	writeConf(t, dir, fmt.Sprintf("# c\n[main]\nrpcport=%d\n", port))
	writeCookie(t, dir)
	d := DetectCore(env(dir, true))
	if d.Mode != ModeFull || d.RPCNotice || d.RPCPort != port || filepath.Base(d.CookiePath) != ".cookie" {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectSyncingCoreLight(t *testing.T) {
	dir := t.TempDir()
	port := rpcServer(t, true, 200)
	writeConf(t, dir, fmt.Sprintf("rpcport=%d\n", port))
	writeCookie(t, dir)
	d := DetectCore(env(dir, true))
	if d.Mode != ModeLight || d.RPCNotice {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectRunningNoCookieNotice(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, dir, fmt.Sprintf("rpcport=%d\n", closedPort(t)))
	d := DetectCore(env(dir, true))
	if d.Mode != ModeLight || !d.RPCNotice {
		t.Fatalf("got %+v", d)
	}
}

func TestDetect401Notice(t *testing.T) {
	dir := t.TempDir()
	port := rpcServer(t, false, 401)
	writeConf(t, dir, fmt.Sprintf("rpcport=%d\n", port))
	writeCookie(t, dir)
	d := DetectCore(env(dir, true))
	if d.Mode != ModeLight || !d.RPCNotice {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectConfDatadir(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	port := rpcServer(t, false, 200)
	writeConf(t, dir, fmt.Sprintf("datadir=%s\nrpcport=%d\n", other, port))
	writeCookie(t, other)
	d := DetectCore(env(dir, true))
	if d.Mode != ModeFull || d.CookiePath != filepath.Join(other, ".cookie") {
		t.Fatalf("got %+v", d)
	}
}

func credServer(t *testing.T, user, pass string) (port int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass || r.Method != "POST" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"result":{"initialblockdownload":false},"error":null,"id":"ocm"}`)
	}))
	t.Cleanup(srv.Close)
	uu, _ := url.Parse(srv.URL)
	port, _ = strconv.Atoi(uu.Port())
	return port
}

func TestDetectConfCredentialsFullMode(t *testing.T) {
	dir := t.TempDir()
	port := credServer(t, "vert", "s3cret")
	writeConf(t, dir, fmt.Sprintf("server=1\nrpcuser=vert\nrpcpassword=s3cret\nrpcport=%d\n", port))
	d := DetectCore(env(dir, true))
	if d.Mode != ModeFull || d.RPCNotice || d.CookiePath != "" || d.RPCUser != "vert" || d.RPCPass != "s3cret" {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectConfCredentialsPreferredOverCookie(t *testing.T) {
	dir := t.TempDir()
	port := credServer(t, "vert", "s3cret")
	writeConf(t, dir, fmt.Sprintf("rpcuser=vert\nrpcpassword=s3cret\nrpcport=%d\n", port))
	writeCookie(t, dir) // stale cookie: the server rejects it
	d := DetectCore(env(dir, true))
	if d.Mode != ModeFull || d.RPCUser != "vert" || d.CookiePath != "" {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectConfCredentialsWrongNotice(t *testing.T) {
	dir := t.TempDir()
	port := credServer(t, "vert", "right")
	writeConf(t, dir, fmt.Sprintf("rpcuser=vert\nrpcpassword=wrong\nrpcport=%d\n", port))
	d := DetectCore(env(dir, true))
	if d.Mode != ModeLight || !d.RPCNotice || d.RPCUser != "" || d.RPCPass != "" {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectRPCAuthOnlyUsesCookie(t *testing.T) {
	dir := t.TempDir()
	port := rpcServer(t, false, 200)
	writeConf(t, dir, fmt.Sprintf("rpcauth=vert:abc$def\nrpcport=%d\n", port))
	writeCookie(t, dir)
	d := DetectCore(env(dir, true))
	if d.Mode != ModeFull || d.RPCUser != "" || filepath.Base(d.CookiePath) != ".cookie" {
		t.Fatalf("got %+v", d)
	}
}

func TestDetectRPCUserWithoutPasswordUsesCookie(t *testing.T) {
	dir := t.TempDir()
	writeConf(t, dir, fmt.Sprintf("rpcuser=vert\nrpcport=%d\n", closedPort(t)))
	d := DetectCore(env(dir, true))
	if d.Mode != ModeLight || !d.RPCNotice || d.RPCUser != "" {
		t.Fatalf("got %+v", d)
	}
}
