package pools

type Pool interface {
	GetPendingPayout() uint64
	GetStratumUrl() string
	GetUsername() string
	GetPassword() string
	GetName() string
	GetID() int
	GetFee() float64
	OpenBrowserPayoutInfo(addr string)
}

// GetPool returns the only pool: the local P2Pool-Go node.
func GetPool(addr string) Pool {
	return NewP2PoolGo(addr)
}
