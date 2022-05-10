package types

import (
	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "vpool"
	StoreKey   = "vpoolkey"
)

/*
PoolKey 			       | 0x00 + PairString 			 | The Pool struct
PoolReserveSnapshotCounter | 0x01 + PairString 			 | Integer
PoolReserveSnapshots       | 0x02 + PairString + Counter | Snapshot
*/
var (
	PoolKey                    = []byte{0x00}
	PoolReserveSnapshotCounter = []byte{0x01}
	PoolReserveSnapshots       = []byte{0x02}
)

// GetPoolKey returns pool key for KVStore
func GetPoolKey(pair common.TokenPair) []byte {
	return append(PoolKey, []byte(pair)...)
}

// GetSnapshotCounterKey returns the KVStore for the Snapshot Pool counters.
func GetSnapshotCounterKey(pair common.TokenPair) []byte {
	return append(PoolReserveSnapshotCounter, []byte(pair)...)
}

// GetSnapshotKey returns the KVStore for the pool reserve snapshots.
func GetSnapshotKey(pair common.TokenPair, counter uint64) []byte {
	return append(
		PoolReserveSnapshots,
		append(
			[]byte(pair),
			sdk.Uint64ToBigEndian(counter)...,
		)...,
	)
}
