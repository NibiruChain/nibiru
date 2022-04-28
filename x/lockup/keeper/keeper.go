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
	// create new lock object
	lock := &types.Lock{
		Owner:    owner.String(),
		Duration: duration,
		EndTime:  ctx.BlockTime().Add(duration),
		Coins:    coins,
	}

	k.LocksState(ctx).Create(lock)
	// move coins from owner to module account
	if err := k.bk.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, coins); err != nil {
		return types.Lock{}, err
	}

	return *lock, nil
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
