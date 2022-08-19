package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ModuleName   = "vpool"
	StoreKey     = "vpoolkey"
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
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
	TWAPPrefix                 = []byte{0x03}
)

// GetPoolKey returns pool key for KVStore
func GetPoolKey(pair common.AssetPair) []byte {
	return append(PoolKey, []byte(pair.String())...)
}

// GetSnapshotCounterKey returns the KVStore for the Snapshot Pool counters.
func GetSnapshotCounterKey(pair common.AssetPair) []byte {
	return append(PoolReserveSnapshotCounter, []byte(pair.String())...)
}

// GetSnapshotKey returns the KVStore for the pool reserve snapshots.
func GetSnapshotKey(pair common.AssetPair, counter uint64) []byte {
	return append(
		PoolReserveSnapshots,
		append(
			[]byte(pair.String()),
			sdk.Uint64ToBigEndian(counter)...,
		)...,
	)
}

// CurrentTWAPKey returns the prefix for the current TWAP price
func CurrentTWAPKey(pair common.AssetPair) []byte {
	twapPairID := "twap-" + pair.String()
	return append(TWAPPrefix, []byte(twapPairID)...)
}
