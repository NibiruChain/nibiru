package keeper_test

import (
	"testing"
	"time"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/lockup/keeper"
	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestLockState(t *testing.T) {
	app, ctx := simapp2.NewTestNibiruAppAndContext(true)
	addr := sample.AccAddress()
	lock := &types.Lock{
		LockId:   0,
		Owner:    addr.String(),
		Duration: 1000 * time.Second,
		EndTime:  ctx.BlockTime(),
		Coins:    sdk.NewCoins(sdk.NewCoin("test", sdk.NewInt(1000))),
	}

	// test create
	app.LockupKeeper.LocksState(ctx).Create(lock)
	// test get
	getLock, err := app.LockupKeeper.LocksState(ctx).Get(keeper.LockStartID) // we're getting the first starting
	require.NoError(t, err)
	require.Equal(t, lock, getLock)
	// test get by addr
	var found = 0
	app.LockupKeeper.LocksState(ctx).IterateLocksByAddress(addr, func(id uint64) (stop bool) {
		iterLock, err := app.LockupKeeper.LocksState(ctx).Get(id)
		require.NoError(t, err)
		require.Equal(t, lock, iterLock)
		found++
		return false
	})
	require.Equal(t, found, 1)
	// test delete
	err = app.LockupKeeper.LocksState(ctx).Delete(getLock)
	require.NoError(t, err)
	// test get not found
	_, err = app.LockupKeeper.LocksState(ctx).Get(getLock.LockId)
	require.ErrorIs(t, err, types.ErrLockupNotFound)
	// test delete not found
	err = app.LockupKeeper.LocksState(ctx).Delete(lock)
	require.ErrorIs(t, err, types.ErrLockupNotFound)
}

func Test_LocksState_232(t *testing.T) {
	// ref: https://github.com/NibiruChain/nibiru/issues/232

	// to simulate this scenario we create two addresses
	// which differentiate from one another by adding +1 to the last byte
	// in such a way that iteration, if done wrong would include both addresses namespaces
	addr1 := sdk.AccAddress{197, 74, 130, 194, 229, 233, 119, 113, 119, 172, 13, 56, 95, 110, 234, 199, 255, 102, 24, 142}
	addr2 := sdk.AccAddress{198, 74, 130, 194, 229, 233, 119, 113, 119, 172, 13, 56, 95, 110, 234, 199, 255, 102, 24, 143}

	app, _ := simapp2.NewTestNibiruAppAndContext(false)
	ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

	// we create two locks for each addr one expired the other one not
	addr1LockedFunds := sdk.NewCoins(sdk.NewInt64Coin("addr1Locked", 500))
	addr1UnlockedFunds := sdk.NewCoins(sdk.NewInt64Coin("addr1Unlocked", 600))
	addr2LockedFunds := sdk.NewCoins(sdk.NewInt64Coin("addr2Locked", 700))
	addr2UnlockedFunds := sdk.NewCoins(sdk.NewInt64Coin("addr2Unlocked", 800))

	// fund both accounts
	require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr1, addr1LockedFunds.Add(addr1UnlockedFunds...).Sort()))
	require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr2, addr2LockedFunds.Add(addr2UnlockedFunds...).Sort()))

	// create locks
	lock1, err := app.LockupKeeper.LockTokens(ctx, addr1, addr1LockedFunds, 2*time.Hour)
	require.NoError(t, err)
	lock2, err := app.LockupKeeper.LockTokens(ctx, addr1, addr1UnlockedFunds, 1*time.Hour)
	require.NoError(t, err)
	_, err = app.LockupKeeper.LockTokens(ctx, addr2, addr2LockedFunds, 2*time.Hour)
	require.NoError(t, err)
	lock4, err := app.LockupKeeper.LockTokens(ctx, addr2, addr2UnlockedFunds, 1*time.Hour)
	require.NoError(t, err)

	// initiate unlock for lock2 and lock4
	_, err = app.LockupKeeper.InitiateUnlocking(ctx, lock2.LockId)
	require.NoError(t, err)
	_, err = app.LockupKeeper.InitiateUnlocking(ctx, lock4.LockId)
	require.NoError(t, err)

	// create a new ctx which is 1hour + 1sec ahead in time of current context, which means
	// lock2 and lock4 will be available

	ctx = app.NewContext(false, tmproto.Header{Time: ctx.BlockTime().Add(1*time.Hour + 1*time.Second)})

	// iterating over addr2 must only return lock4, note: we iterate over addr2 because it's bigger than addr1
	// since unlocked coins goes backwards from addr2-blockTime, if the impl was wrong then addr1 both locked
	// and unlocked coins would be included.
	coins, err := app.LockupKeeper.AccountUnlockedCoins(ctx, addr2)
	require.NoError(t, err)
	require.Equal(t, coins, lock4.Coins)

	// iterating over addr1 locked coins must return only lock1, note: we iterate over addr1 because it's smaller
	// than addr2, since locked coins goes from addr1-blockTime, if the impl was wrong then addr2 both locked
	// and unlocked coins would be included.
	coins, err = app.LockupKeeper.AccountLockedCoins(ctx, addr1)
	require.NoError(t, err)
	require.Equal(t, coins, lock1.Coins)
}
