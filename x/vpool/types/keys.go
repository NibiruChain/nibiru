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
PoolKeyPrefix 			 | 0x00 + PairString 			   | The Pool struct
SnapshotsKeyPrefix       | 0x01 + PairString + BlockHeight | Snapshot
*/
var (
	PoolKeyPrefix      = []byte{0x00}
	SnapshotsKeyPrefix = []byte{0x01}
)

// GetPoolKey returns pool key for KVStore
func GetPoolKey(pair common.AssetPair) []byte {
	return append(PoolKeyPrefix, []byte(pair.String())...)
}

// GetSnapshotKey returns the KVStore for the pool reserve snapshots.
func GetSnapshotKey(pair common.AssetPair, blockHeight uint64) []byte {
	return append(
		SnapshotsKeyPrefix,
		append(
			[]byte(pair.String()),
			sdk.Uint64ToBigEndian(blockHeight)...,
		)...,
	)
}
