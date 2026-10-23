package wallet

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/vertiond/verthash-one-click-miner/networks"
)

func useServers(t *testing.T, handlers ...http.HandlerFunc) []*int32 {
	t.Helper()
	hits := make([]*int32, len(handlers))
	urls := make([]string, len(handlers))
	for i, h := range handlers {
		hits[i] = new(int32)
		counter, handler := hits[i], h
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(counter, 1)
			handler(w, r)
		}))
		t.Cleanup(srv.Close)
		urls[i] = srv.URL
	}
	old := networks.Active.BlockbookURLs
	networks.Active.BlockbookURLs = urls
	t.Cleanup(func() { networks.Active.BlockbookURLs = old })
	return hits
}

func TestUtxosSplitsImmatureCoinbase(t *testing.T) {
	useServers(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/utxo/DTestAddress" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		io.WriteString(w, `[
			{"txid":"aa","vout":0,"value":"100000000","confirmations":5},
			{"txid":"bb","vout":1,"value":"250000000","confirmations":300,"coinbase":true},
			{"txid":"cc","vout":2,"value":"700000000","confirmations":239,"coinbase":true},
			{"txid":"dd","vout":3,"value":"5","confirmations":240,"coinbase":true}
		]`)
	})

	spendable, immature, err := blockbookUtxos("DTestAddress")
	if err != nil {
		t.Fatal(err)
	}
	if len(spendable) != 3 || spendable[0].TxID != "aa" || spendable[1].TxID != "bb" || spendable[2].TxID != "dd" {
		t.Fatalf("unexpected spendable outputs: %+v", spendable)
	}
	if spendable[1].Vout != 1 || spendable[1].Amount != 250000000 {
		t.Fatalf("unexpected output values: %+v", spendable[1])
	}
	if immature != 700000000 {
		t.Fatalf("immature = %d, want 700000000", immature)
	}
}

func TestRequestFailsOverToNextServer(t *testing.T) {
	hits := useServers(t,
		func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusBadGateway) },
		func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `[]`) },
	)

	if _, _, err := blockbookUtxos("DTestAddress"); err != nil {
		t.Fatal(err)
	}
	if *hits[0] != 1 || *hits[1] != 1 {
		t.Fatalf("hits = %d, %d; want 1, 1", *hits[0], *hits[1])
	}
}

func TestRequestFailsWhenAllServersFail(t *testing.T) {
	useServers(t,
		func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusBadGateway) },
		func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `not json`) },
	)

	if _, _, err := blockbookUtxos("DTestAddress"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSendPostsRawHexAndReturnsTxid(t *testing.T) {
	var body string
	useServers(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/sendtx/" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		io.WriteString(w, `{"result":"deadbeef"}`)
	})

	txid, err := blockbookSend("0100aabb")
	if err != nil {
		t.Fatal(err)
	}
	if txid != "deadbeef" || body != "0100aabb" {
		t.Fatalf("txid = %q, body = %q", txid, body)
	}
}

func TestRejectedTransactionIsNotRetriedElsewhere(t *testing.T) {
	hits := useServers(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"-26: dust"}`)
		},
		func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"result":"should-not-happen"}`) },
	)

	if _, err := blockbookSend("00"); err == nil {
		t.Fatal("expected the rejection to be returned")
	}
	if *hits[1] != 0 {
		t.Fatalf("second server was asked %d times, want 0", *hits[1])
	}
}

func TestSendFailsOverOnServerError(t *testing.T) {
	useServers(t,
		func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) },
		func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"result":"cafe"}`) },
	)

	txid, err := blockbookSend("00")
	if err != nil || txid != "cafe" {
		t.Fatalf("txid = %q, err = %v", txid, err)
	}
}
