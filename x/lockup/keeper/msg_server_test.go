package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/lockup/keeper"
	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestMsgServer_LockTokens(t *testing.T) {
	app := testutil.NewTestApp(false)
	uncachedCtx := app.NewContext(false, tmproto.Header{Time: time.Now()})
	s := keeper.NewMsgServerImpl(app.LockupKeeper)

	t.Run("success", func(t *testing.T) {
		ctx, _ := uncachedCtx.CacheContext()
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		_, err := s.LockTokens(sdk.WrapSDKContext(ctx), &types.MsgLockTokens{
			Owner:    addr.String(),
			Duration: 0,
			Coins:    nil,
		})
		require.NoError(t, err)
	})
}

func TestMsgServer_InitiateUnlock(t *testing.T) {
	app := testutil.NewTestApp(false)
	uncachedCtx := app.NewContext(false, tmproto.Header{Time: time.Now()})
	s := keeper.NewMsgServerImpl(app.LockupKeeper)

	t.Run("success", func(t *testing.T) {
		ctx, _ := uncachedCtx.CacheContext()
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		lock, err := s.LockTokens(sdk.WrapSDKContext(ctx), &types.MsgLockTokens{
			Owner:    addr.String(),
			Duration: 0,
			Coins:    coins,
		})
		require.NoError(t, err)

		_, err = s.InitiateUnlock(sdk.WrapSDKContext(ctx), &types.MsgInitiateUnlock{
			Owner:  addr.String(),
			LockId: lock.LockId,
		})
		require.NoError(t, err)
	})
}

func TestQueryServer_Lock(t *testing.T) {
	app := testutil.NewTestApp(false)
	uncachedCtx := app.NewContext(false, tmproto.Header{Time: time.Now()})
	s := keeper.NewMsgServerImpl(app.LockupKeeper)
	q := keeper.NewQueryServerImpl(app.LockupKeeper)

	t.Run("success", func(t *testing.T) {
		ctx, _ := uncachedCtx.CacheContext()
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		lockID, err := s.LockTokens(sdk.WrapSDKContext(ctx), &types.MsgLockTokens{
			Owner:    addr.String(),
			Duration: 0,
			Coins:    coins,
		})
		require.NoError(t, err)

		// query lock
		resp, err := q.Lock(sdk.WrapSDKContext(ctx), &types.QueryLockRequest{Id: lockID.LockId})
		require.NoError(t, err)

		require.Equal(t, lockID.LockId, resp.Lock.LockId)
		require.Equal(t, coins, resp.Lock.Coins)
		require.Equal(t, addr.String(), resp.Lock.Owner)
	})
}

func TestQueryServer_LockedCoins(t *testing.T) {
	app := testutil.NewTestApp(false)
	uncachedCtx := app.NewContext(false, tmproto.Header{Time: time.Now()})
	s := keeper.NewMsgServerImpl(app.LockupKeeper)
	q := keeper.NewQueryServerImpl(app.LockupKeeper)

	t.Run("success", func(t *testing.T) {
		ctx, _ := uncachedCtx.CacheContext()
		addr := sample.AccAddress()
		coins := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

		_, err := s.LockTokens(sdk.WrapSDKContext(ctx), &types.MsgLockTokens{
			Owner:    addr.String(),
			Duration: 0,
			Coins:    coins,
		})
		require.NoError(t, err)

		res, err := q.LockedCoins(sdk.WrapSDKContext(ctx), &types.QueryLockedCoinsRequest{Address: addr.String()})
		require.NoError(t, err)

		require.Equal(t, coins, res.LockedCoins)
	})
}

func TestQueryServer_LocksByAddress(t *testing.T) {
	app := testutil.NewTestApp(false)
	uncachedCtx := app.NewContext(false, tmproto.Header{Time: time.Now()})
	s := keeper.NewMsgServerImpl(app.LockupKeeper)
	q := keeper.NewQueryServerImpl(app.LockupKeeper)
	t.Run("success", func(t *testing.T) {
		ctx, _ := uncachedCtx.CacheContext()
		addr := sample.AccAddress()
		totalQuery := 50
		totalFromQuery := sdk.NewCoins()
		// create locks
		for i := 0; i < 100; i++ {
			coins := sdk.NewCoins(sdk.NewInt64Coin("test", 100+int64(i)))
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, coins))

			_, err := s.LockTokens(sdk.WrapSDKContext(ctx), &types.MsgLockTokens{
				Owner:    addr.String(),
				Duration: 1 + time.Duration(i)*time.Hour,
				Coins:    coins,
			})
			require.NoError(t, err)

			// lets compute total coins
			if i < totalQuery {
				totalFromQuery = totalFromQuery.Add(coins...)
			}
		}

		// query
		res, err := q.LocksByAddress(sdk.WrapSDKContext(ctx), &types.QueryLocksByAddress{
			Address: addr.String(),
			Pagination: &query.PageRequest{
				Offset: 0,
				Limit:  uint64(totalQuery),
			},
		})
		require.NoError(t, err)

		require.Len(t, res.Locks, totalQuery)

		totalFromResponse := sdk.NewCoins()
		for _, l := range res.Locks {
			totalFromResponse = totalFromResponse.Add(l.Coins...)
		}

		require.Equal(t, totalFromQuery, totalFromResponse)
	})
}
