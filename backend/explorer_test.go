package backend

import (
	"testing"

	"github.com/vertiond/verthash-one-click-miner/networks"
)

func TestExplorerTxURLPointsAtDogechain(t *testing.T) {
	networks.SetNetwork(false)
	txid := "cfb6b8d5ff5a2b3d0c9321db11ea90e96bfd0ea6fe6f5bc06e3e8c6f29a714cd"
	want := "https://dogechain.com/tx/" + txid
	if got := explorerTxURL(txid); got != want {
		t.Fatalf("explorerTxURL() = %q, want %q", got, want)
	}
}
