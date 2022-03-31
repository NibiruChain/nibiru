package keeper

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/MatrixDao/matrix/x/lockup/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LockupKeeper provides a way to manage module storage.
type LockupKeeper struct {
	cdc      codec.Codec
	storeKey sdk.StoreKey

	ak types.AccountKeeper
	bk types.BankKeeper
	dk types.DistrKeeper
}

// NewKeeper returns an instance of Keeper.
func NewLockupKeeper(cdc codec.Codec, storeKey sdk.StoreKey, ak types.AccountKeeper,
	bk types.BankKeeper, dk types.DistrKeeper) LockupKeeper {
	return LockupKeeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
		bk:       bk,
		dk:       dk,
	}
}

// Logger returns a logger instance.
func (k LockupKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// LockTokens lock tokens from an account for specified duration.
func (k LockupKeeper) LockTokens(ctx sdk.Context, owner sdk.AccAddress,
	coins sdk.Coins, duration time.Duration) (types.Lock, error) {
	lockId := k.GetLastLockId(ctx) + 1
	// unlock time is set at the beginning of unlocking time
	lock := types.NewLock(lockId, owner, duration, time.Time{}, coins)
	err := k.lock(ctx, lock)
	if err != nil {
		return lock, err
	}
	k.SetLastLockId(ctx, lockId)
	return lock, nil
}

// Lock is a utility to lock coins into module account.
func (k LockupKeeper) lock(ctx sdk.Context, lock types.Lock) error {
	owner, err := sdk.AccAddressFromBech32(lock.Owner)
	if err != nil {
		return err
	}
	if err := k.bk.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, lock.Coins); err != nil {
		return err
	}

	// store lock object into the store
	store := ctx.KVStore(k.storeKey)
	bz, err := lock.Marshal()
	if err != nil {
		return err
	}
	store.Set(lockStoreKey(lock.LockId), bz)

	return nil
}
