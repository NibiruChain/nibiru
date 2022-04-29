package keeper

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// MaxTime is the maximum golang time that can be represented.
	// It is a date so forward in the future that identifies a
	// lockup which did not yet start to unlock.
	// NOTE: this is not maximum golang time because, since we encode it
	// using timestampb.Timestamp under the hood we need to adhere to proto time
	// rules. Equivalent to: time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
	MaxTime = time.Unix(253402297199, 0).UTC()
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
	coins sdk.Coins, duration time.Duration) (*types.Lock, error) {
	// create new lock object
	lock := &types.Lock{
		Owner:    owner.String(),
		Duration: duration,
		EndTime:  MaxTime, // we set MaxTime here which means not unlocking.
		Coins:    coins,
	}

	k.LocksState(ctx).Create(lock)
	// move coins from owner to module account
	if err := k.bk.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, coins); err != nil {
		return nil, err
	}

	return lock, nil
}

// UnlockTokens returns tokens back from the module account address to the lock owner.
// The ID associated with the lock must exist, and the current block time must be after lock end time.
func (k LockupKeeper) UnlockTokens(ctx sdk.Context, lockID uint64) (unlockedTokens sdk.Coins, err error) {
	lock, err := k.LocksState(ctx).Get(lockID)
	if err != nil {
		return nil, err
	}

	// assert time constraints have been met
	if currentTime := ctx.BlockTime(); currentTime.Before(lock.EndTime) {
		return unlockedTokens, types.ErrLockEndTime.Wrapf("current time: %s, end time: %s", currentTime, lock.EndTime)
	}

	owner, err := sdk.AccAddressFromBech32(lock.Owner)
	if err != nil {
		panic(err)
	}

	if err = k.bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, lock.Coins); err != nil {
		panic(err) // invariant broken: module account holds fewer coins compared to what can be withdrawn
	}

	// cleanup state
	err = k.LocksState(ctx).Delete(lock)
	if err != nil {
		panic(err) // invariant broken, delete must never fail after get
	}

	return lock.Coins, nil
}

// InitiateUnlocking starts the unlocking process of a lockup.
func (k LockupKeeper) InitiateUnlocking(ctx sdk.Context, lockID uint64) (updatedLock *types.Lock, err error) {
	// we get the lockup
	lock, err := k.LocksState(ctx).Get(lockID)
	if err != nil {
		return nil, err
	}

	// we check if unlocking did not yet start
	if !lock.EndTime.Equal(MaxTime) {
		return nil, types.ErrAlreadyUnlocking.Wrapf("unlock for lock %d was already initiated and will mature at %s", lockID, lock.EndTime)
	}

	// if it did not yet start we update the lock's end time
	// which initiates the unlocking of the assets.
	lock.EndTime = ctx.BlockTime().Add(lock.Duration)
	err = k.LocksState(ctx).Update(lock)
	if err != nil {
		panic(err)
	}

	return lock, nil
}

// UnlockAvailableCoins unlocks all the available coins for the provided account sdk.AccAddress.
func (k LockupKeeper) UnlockAvailableCoins(ctx sdk.Context, account sdk.AccAddress) (coins sdk.Coins, err error) {
	ids := k.LocksState(ctx).UnlockedIDsByAddress(account)

	coins = sdk.NewCoins()
	for _, id := range ids {
		unlockedCoins, err := k.UnlockTokens(ctx, id)
		if err != nil {
			panic(fmt.Errorf("state corruption: %w", err))
		}

		coins = coins.Add(unlockedCoins...)
	}

	return coins, nil
}

// AccountLockedCoins returns the locked coins of the given sdk.AccAddress
func (k LockupKeeper) AccountLockedCoins(ctx sdk.Context, account sdk.AccAddress) (coins sdk.Coins, err error) {
	return k.LocksState(ctx).IterateLockedCoins(account), nil
}

// AccountUnlockedCoins returns the unlocked coins of the given sdk.AccAddress
func (k LockupKeeper) AccountUnlockedCoins(ctx sdk.Context, account sdk.AccAddress) (coins sdk.Coins, err error) {
	return k.LocksState(ctx).IterateUnlockedCoins(account), nil
}

// TotalLockedCoins returns the module account locked coins.
func (k LockupKeeper) TotalLockedCoins(ctx sdk.Context) (coins sdk.Coins, err error) {
	return k.LocksState(ctx).IterateTotalLockedCoins(), nil
}

// LocksByDenom allows to iterate over types.Lock associated with a denom.
// CONTRACT: no writes on store can happen until the function exits.
func (k LockupKeeper) LocksByDenom(ctx sdk.Context, do func(lock *types.Lock) (stop bool)) (coins sdk.Coins, err error) {
	panic("impl")
}

// LocksByDenomUnlockingAfter allows to iterate over types.Lock associated with a denom that unlock
// after the provided duration.
// CONTRACT: no writes on store can happen until the function exits.
func (k LockupKeeper) LocksByDenomUnlockingAfter(ctx sdk.Context, denom string, duration time.Duration, do func(lock *types.Lock) (stop bool)) {
	endTime := ctx.BlockTime().Add(duration)
	state := k.LocksState(ctx)
	state.IterateCoinsByDenomUnlockingAfter(denom, endTime, func(id uint64) (stop bool) {
		lock, err := state.Get(id)
		if err != nil {
			panic(err)
		}

		return do(lock)
	})
}
