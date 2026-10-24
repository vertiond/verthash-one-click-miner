package util

import (
	"math/big"
	"testing"
)

func TestNetHashFromDifficulty(t *testing.T) {
	difficulty := big.NewFloat(24.8)
	factor := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(48), nil)
	netHash := difficulty.Mul(difficulty, big.NewFloat(0).SetInt(factor))
	want, _ := netHash.Quo(netHash, big.NewFloat(9830250)).Uint64()
	if got := NetHashFromDifficulty(24.8); got != want || got == 0 {
		t.Fatalf("got %d want %d", got, want)
	}
}
