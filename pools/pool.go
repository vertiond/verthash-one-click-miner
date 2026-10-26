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

func GetPools(addr string) []Pool {
	return []Pool{
		NewZergpool(addr),
		NewSuprnova(addr),
		NewP2Pool(addr),
		Newzpool(addr),
		NewMiningpoolSweden(addr),
	}
}

func GetPool(pool int, addr string) Pool {
	pools := GetPools(addr)
	for _, p := range pools {
		if p.GetID() == pool {
			return p
		}
	}
	return pools[0]
}
