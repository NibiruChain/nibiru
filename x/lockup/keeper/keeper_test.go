package keeper_test

import (
	"testing"
	"time"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCreateLock(t *testing.T) {
	tests := []struct {
		name                string
		accountInitialFunds sdk.Coins
		ownerAddr           sdk.AccAddress
		coins               sdk.Coins
		duration            time.Duration
		shouldErr           bool
	}{
		{
			name:                "happy path",
			accountInitialFunds: sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			ownerAddr:           sample.AccAddress(),
			coins:               sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			duration:            24 * time.Hour,
			shouldErr:           false,
		},
		{
			name:                "not enough funds",
			accountInitialFunds: sdk.NewCoins(sdk.NewInt64Coin("foo", 99)),
			ownerAddr:           sample.AccAddress(),
			coins:               sdk.NewCoins(sdk.NewInt64Coin("foo", 100)),
			duration:            24 * time.Hour,
			shouldErr:           true,
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, tc.ownerAddr, tc.accountInitialFunds))

			lock, err := app.LockupKeeper.LockTokens(ctx, tc.ownerAddr, tc.coins, tc.duration)
			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				require.Equal(t, types.Lock{
					LockId:   0,
					Owner:    tc.ownerAddr.String(),
					Duration: tc.duration,
					Coins:    tc.coins,
					EndTime:  ctx.BlockTime().Add(24 * time.Hour),
				}, lock)
			}
		})
	}
}

func TestLockupKeeper_UnlockTokens(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, _ := testutil.NewNibiruApp(true)
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewCoin("test", sdk.NewInt(1000)))

		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		lock, err := app.LockupKeeper.LockTokens(ctx, addr, coins, time.Second*1000)
		require.NoError(t, err)

		// unlock coins
		ctx = app.NewContext(false, tmproto.Header{Time: ctx.BlockTime().Add(lock.Duration + 1*time.Second)}) // instantiate a new context with oldBlockTime+lock duration+1 second

		unlockedCoins, err := app.LockupKeeper.UnlockTokens(ctx, lock.LockId)
		require.NoError(t, err)

		require.Equal(t, lock.Coins, unlockedCoins)

		// assert cleanups
		_, err = app.LockupKeeper.UnlockTokens(ctx, lock.LockId)
		require.ErrorIs(t, err, types.ErrLockupNotFound)
	})

	t.Run("lock not found", func(t *testing.T) {
		app, ctx := testutil.NewNibiruApp(true)

		_, err := app.LockupKeeper.UnlockTokens(ctx, 1)
		require.ErrorIs(t, err, types.ErrLockupNotFound)
	})

	t.Run("lock not matured", func(t *testing.T) {
		app, _ := testutil.NewNibiruApp(true)
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewCoin("test", sdk.NewInt(1000)))

		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		lock, err := app.LockupKeeper.LockTokens(ctx, addr, coins, time.Second*1000)
		require.NoError(t, err)

		_, err = app.LockupKeeper.UnlockTokens(ctx, lock.LockId) // we use the same ctx which means lock up duration did not mature yet
		require.ErrorIs(t, err, types.ErrLockEndTime)
	})
}

func TestLockupKeeper_AccountLockedCoins(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, _ := testutil.NewNibiruApp(true)
		addr := sample.AccAddress()
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		// 1st lock
		coins1 := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins1))
		_, err := app.LockupKeeper.LockTokens(ctx, addr, coins1, time.Second*1000)
		require.NoError(t, err)

		// 2nd lock
		coins2 := sdk.NewCoins(sdk.NewCoin("osmo", sdk.NewInt(10000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins2))
		_, err = app.LockupKeeper.LockTokens(ctx, addr, coins2, time.Second*1500)
		require.NoError(t, err)

		// query locks
		lockedCoins, err := app.LockupKeeper.AccountLockedCoins(ctx, addr)
		require.NoError(t, err)

		require.Equal(t, lockedCoins, coins1.Add(coins2...).Sort())
	})
}

func TestLockupKeeper_AccountUnlockedCoins(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, _ := testutil.NewNibiruApp(false)
		addr := sample.AccAddress()
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		// 1st lock
		coins1 := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins1))
		_, err := app.LockupKeeper.LockTokens(ctx, addr, coins1, time.Second*1000)
		require.NoError(t, err)

		// 2nd lock
		coins2 := sdk.NewCoins(sdk.NewCoin("osmo", sdk.NewInt(10000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins2))
		_, err = app.LockupKeeper.LockTokens(ctx, addr, coins2, time.Second*1500)
		require.NoError(t, err)

		// query unlocked coins
		ctx = app.NewContext(false, tmproto.Header{Time: time.Now().Add(1100 * time.Second)}) // we create a new context in which only the first coins are unlocked
		unlockedCoins, err := app.LockupKeeper.AccountUnlockedCoins(ctx, addr)
		require.NoError(t, err)

		require.Equal(t, unlockedCoins, coins1)
	})
}

func TestLockupKeeper_LockedCoins(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, _ := testutil.NewNibiruApp(false)
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		addr := sample.AccAddress()
		// 1st lock which will become unlocked
		coinsThatUnlock := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coinsThatUnlock))
		_, err := app.LockupKeeper.LockTokens(ctx, addr, coinsThatUnlock, time.Second*1)
		require.NoError(t, err)

		// 2nd lock which is locked in this test case
		coinsThatRemainLocked := sdk.NewCoins(sdk.NewCoin("osmo", sdk.NewInt(10000)))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coinsThatRemainLocked))
		_, err = app.LockupKeeper.LockTokens(ctx, addr, coinsThatRemainLocked, time.Second*1500)
		require.NoError(t, err)

		ctx = app.NewContext(false, tmproto.Header{Time: ctx.BlockTime().Add(10 * time.Second)}) // new context 10 seconds forward which means only 1 set is unlocked

		gotLockedCoins, err := app.LockupKeeper.TotalLockedCoins(ctx)
		require.NoError(t, err)

		require.Equal(t, gotLockedCoins, coinsThatRemainLocked)
	})
}
