package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	ModuleName = "vamm"
	StoreKey   = "ammkey"
)

/*
PoolKey 			       | 0x00 + PairString | The Pool struct
PoolReserveSnapshotCounter | 0x01 + PairString | Integer
PoolReserveSnapshots       | 0x02 + Counter    | Snapshot
*/
var (
	PoolKey                    = []byte{0x00}
	PoolReserveSnapshotCounter = []byte{0x01}
	PoolReserveSnapshots       = []byte{0x02}
)

// GetPoolKey returns pool key for KVStore
func GetPoolKey(pair string) []byte {
	return append(PoolKey, []byte(pair)...)
}

// GetPoolReserveSnapshotCounter returns the KVStore for the Snapshot Pool counters.
func GetPoolReserveSnapshotCounter(pair string) []byte {
	return append(PoolReserveSnapshotCounter, []byte(pair)...)
}

// GetPoolReserveSnapshotKey returns the KVStore for the pool reserve snapshots.
func GetPoolReserveSnapshotKey(pair string, counter int64) []byte {
	return append(PoolReserveSnapshots, sdk.Uint64ToBigEndian(uint64(counter))...)
}
