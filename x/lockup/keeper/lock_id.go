package keeper

import (
	"github.com/MatrixDao/matrix/x/lockup/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetLastLockId returns ID used last time.
func (k LockupKeeper) GetNextLockId(ctx sdk.Context) (lockId uint64) {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyLastLockId)
	if bz == nil {
		// If uninitialized, start using lockId=0
		lockId = 0
	} else {
		lockId = sdk.BigEndianToUint64(bz)
	}

	// Increment so that next call receives a new number
	store.Set(types.KeyLastLockId, sdk.Uint64ToBigEndian(lockId+1))

	return lockId
}
