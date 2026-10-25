package backend

import "testing"

// A second copy has no settings database and never picks a pool, but its
// mining screen still asks for the pool and payout details.
func TestSecondCopyMiningScreenGettersDontPanic(t *testing.T) {
	b, err := NewBackend(true)
	if err != nil {
		t.Fatalf("NewBackend(true) error = %v", err)
	}
	for name, get := range map[string]func() string{
		"GetPoolName":      b.GetPoolName,
		"GetPoolFee":       b.GetPoolFee,
		"GetPayoutTicker":  b.GetPayoutTicker,
		"DogeOnlyPoolName": b.DogeOnlyPoolName,
	} {
		if got := get(); got != "" {
			t.Errorf("%s() = %q, want empty", name, got)
		}
	}
	if b.PayoutShortage() {
		t.Error("PayoutShortage() = true, want false")
	}
}
