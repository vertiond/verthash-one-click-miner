package wallet

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vertcoin-project/one-click-miner-vnext/networks"
)

func useBackend(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	old := networks.Active.OCMBackend
	networks.Active.OCMBackend = srv.URL + "/"
	t.Cleanup(func() {
		srv.Close()
		networks.Active.OCMBackend = old
	})
}

func TestUpdateReadsBalance(t *testing.T) {
	useBackend(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"confirmed":481776,"maturing":1000}`))
	})
	w := &Wallet{Script: []byte{0x00, 0x14}}
	if err := w.Update(); err != nil {
		t.Fatal(err)
	}
	if b, bi := w.GetBalance(); b != 481776 || bi != 1000 {
		t.Errorf("balance = %d, %d", b, bi)
	}
}

func TestUpdateFailureKeepsLastBalanceAndReportsIt(t *testing.T) {
	useBackend(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	})
	w := &Wallet{Script: []byte{0x00, 0x14}, Spendable: 7, Maturing: 3}
	if err := w.Update(); err == nil {
		t.Fatal("expected an error when the backend fails")
	}
	if b, bi := w.GetBalance(); b != 7 || bi != 3 {
		t.Errorf("balance changed on failure: %d, %d", b, bi)
	}
}
