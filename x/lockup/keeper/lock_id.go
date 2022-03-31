package keeper

import (
	"github.com/MatrixDao/matrix/x/lockup/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetLastLockId returns ID used last time.
func (k LockupKeeper) GetLastLockId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyLastLockId)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// SetLastLockId save ID used by last lock.
func (k LockupKeeper) SetLastLockId(ctx sdk.Context, lockId uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyLastLockId, sdk.Uint64ToBigEndian(lockId))
}
