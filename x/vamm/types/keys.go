package types

const (
	ModuleName = "vamm"
	StoreKey   = "ammkey"
)

/*
PoolKey | 0x00 + PairString | The Pool struct
*/
var (
	PoolKey = []byte{0x00}
)

// GetPoolKey returns pool key for KVStore
func GetPoolKey(pair string) []byte {
	return append(PoolKey, []byte(pair)...)
}
