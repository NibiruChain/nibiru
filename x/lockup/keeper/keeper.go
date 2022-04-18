package keeper

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/lockup/types"
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

// NewLockupKeeper returns an instance of Keeper.
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
	lockId := k.GetNextLockId(ctx)
	// unlock time is set at the beginning of unlocking time
	lock := types.NewLock(lockId, owner, duration, ctx.BlockTime().Add(duration), coins)
	if err := k.lock(ctx, lock); err != nil {
		return lock, err
	}

	return lock, nil
}

// UnlockTokens returns tokens back from the module account address to the lock owner.
// The ID associated with the lock must exist, and the current block time must be after lock end time.
func (k LockupKeeper) UnlockTokens(ctx sdk.Context, lockID uint64) (unlockedTokens sdk.Coins, err error) {
	lockIDKey := lockStoreKey(lockID)
	store := ctx.KVStore(k.storeKey)

	lockBytes := store.Get(lockIDKey)
	if lockBytes == nil {
		return unlockedTokens, types.ErrLockupNotFound.Wrapf("%d", lockID)
	}

	lock := new(types.Lock)
	k.cdc.MustUnmarshal(lockBytes, lock)

	// assert time constraints have been met
	if currentTime := ctx.BlockTime(); currentTime.Before(lock.EndTime) {
		return unlockedTokens, types.ErrLockEndTime.Wrapf("current time: %s, end time: %s", currentTime, lock.EndTime)
	}

	owner, err := sdk.AccAddressFromBech32(lock.Owner)
	if err != nil {
		panic(err)
	}

	if err := k.bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, lock.Coins); err != nil {
		panic(err) // invariant broken: module account holds fewer coins compared to what can be withdrawn
	}

	// cleanup state
	store.Delete(lockIDKey)

	return lock.Coins, nil
}

// lock is a utility to lock coins into module account.
func (k LockupKeeper) lock(ctx sdk.Context, lock types.Lock) (err error) {
	owner, err := sdk.AccAddressFromBech32(lock.Owner)
	if err != nil {
		return err
	}
	if err = k.bk.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, lock.Coins); err != nil {
		return err
	}

	// store lock object into the store
	ctx.KVStore(k.storeKey).Set(lockStoreKey(lock.LockId), k.cdc.MustMarshal(&lock))

	return nil
}
