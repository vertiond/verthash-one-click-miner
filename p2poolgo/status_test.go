package p2poolgo

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const sampleStatus = `{"ready":true,"state":"ready","version":"1.2.3","has_template":true,"synced":true,"peers":7,"shares_total":10,"height":4123456,"difficulty":123.5,"share_difficulty":1.5,"block_reward":5,"expected_payout":0.1,"expected_payout_sats":1000000,"payouts":{},"stratum_port":9172,"p2p_port":9348,"network":"mainnet","testnet":false,"uptime_seconds":4242}`

func TestGetStatusParses(t *testing.T) {
	var gotAddr, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAddr = r.URL.Query().Get("address")
		gotPath = r.URL.Path
		w.Write([]byte(sampleStatus))
	}))
	defer srv.Close()
	s, err := GetStatus(srv.URL, "Vabc123")
	if err != nil {
		t.Fatal(err)
	}
	if gotAddr != "Vabc123" || gotPath != "/api/status" {
		t.Fatalf("server saw %q %q", gotPath, gotAddr)
	}
	want := Status{true, "ready", "1.2.3", 7, 4123456, 123.5, 1000000, 4242}
	if *s != want {
		t.Fatalf("got %+v want %+v", *s, want)
	}
}

func TestGetStatusRejectsNonP2PoolGo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"foo":1}`))
	}))
	defer srv.Close()
	if _, err := GetStatus(srv.URL, ""); !errors.Is(err, ErrNotP2PoolGo) {
		t.Fatalf("got %v", err)
	}
}

func uptimeServer(ups ...int) *httptest.Server {
	var n int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(atomic.AddInt32(&n, 1)) - 1
		if i >= len(ups) {
			i = len(ups) - 1
		}
		fmt.Fprintf(w, `{"state":"ready","version":"1","uptime_seconds":%d}`, ups[i])
	}))
}

func TestDetectExistingAdoptsLiveNode(t *testing.T) {
	srv := uptimeServer(100, 101)
	defer srv.Close()
	if !DetectExisting(srv.URL, 10*time.Millisecond) {
		t.Fatal("expected true")
	}
}

func TestDetectExistingRejectsStaticUptime(t *testing.T) {
	srv := uptimeServer(100, 100)
	defer srv.Close()
	if DetectExisting(srv.URL, 10*time.Millisecond) {
		t.Fatal("expected false")
	}
}

func TestDetectExistingRejectsNonP2PoolGo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>hi</html>"))
	}))
	defer srv.Close()
	if DetectExisting(srv.URL, 10*time.Millisecond) {
		t.Fatal("expected false")
	}
}

func TestDetectExistingNothingListening(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	if DetectExisting("http://"+addr, 10*time.Millisecond) {
		t.Fatal("expected false")
	}
}
