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
	SnapshotsKeyPrefix = []byte{0x01}
)

// GetSnapshotKey returns the composite key for the vpool snapshots.
// Note that it's up to the caller to create the prefix store with SnapshotsPrefixKey
func GetSnapshotKey(pair common.AssetPair, blockHeight uint64) []byte {
	return append(
		[]byte(pair.String()),
		sdk.Uint64ToBigEndian(blockHeight)...,
	)
}
