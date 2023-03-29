package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestGetAndSetNextPoolNumber(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	// Write to store
	app.SpotKeeper.SetNextPoolNumber(ctx, 150)

	// Read from store
	poolNumber, err := app.SpotKeeper.GetNextPoolNumber(ctx)
	assert.NoError(t, err)
	require.EqualValues(t, poolNumber, 150)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	// Write a pool number
	app.SpotKeeper.SetNextPoolNumber(ctx, 200)

	// Get next and increment should return the current pool number
	poolNumber, err := app.SpotKeeper.GetNextPoolNumberAndIncrement(ctx)
	assert.NoError(t, err)
	require.EqualValues(t, poolNumber, 200)

	// Check that the previous call incremented the number
	poolNumber, err = app.SpotKeeper.GetNextPoolNumber(ctx)
	assert.NoError(t, err)
	require.EqualValues(t, poolNumber, 201)
}

func TestSetAndFetchPool(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	pool := types.Pool{
		Id: 150,
		PoolParams: types.PoolParams{
			SwapFee:  sdk.NewDecWithPrec(3, 2),
			ExitFee:  sdk.NewDecWithPrec(3, 2),
			PoolType: types.PoolType_BALANCER,
			A:        sdk.ZeroInt(),
		},
		PoolAssets: []types.PoolAsset{
			{
				Token:  sdk.NewCoin(denoms.BTC, sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
			{
				Token:  sdk.NewCoin(denoms.NUSD, sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
		},
		TotalWeight: sdk.NewInt(2),
		TotalShares: sdk.NewInt64Coin("nibiru/pool/150", 100),
	}

	app.SpotKeeper.SetPool(ctx, pool)

	retrievedPool, err := app.SpotKeeper.FetchPool(ctx, 150)
	assert.NoError(t, err)
	require.Equal(t, pool, retrievedPool)
}

func TestFetchPoolFromPair(t *testing.T) {
	tests := []struct {
		name           string
		firstToken     string
		secondToken    string
		expectedPass   bool
		expectedPoolId uint64
	}{
		{
			name:           "Correct parse pool 1",
			firstToken:     "tokenA",
			secondToken:    "tokenB",
			expectedPass:   true,
			expectedPoolId: 1,
		},
		{
			name:           "Correct parse pool 1 inverted",
			firstToken:     "tokenB",
			secondToken:    "tokenA",
			expectedPass:   true,
			expectedPoolId: sdk.NewInt(1).Uint64(),
		},
		{
			name:           "Correct parse pool 2",
			firstToken:     "tokenB",
			secondToken:    "tokenC",
			expectedPass:   true,
			expectedPoolId: sdk.NewInt(2).Uint64(),
		},
		{
			name:           "Correct parse pool 2 inverted",
			firstToken:     "tokenC",
			secondToken:    "tokenB",
			expectedPass:   true,
			expectedPoolId: sdk.NewInt(2).Uint64(),
		},
		{
			name:           "Correct parse pool 2 inverted",
			firstToken:     "tokenC",
			secondToken:    "tokenB",
			expectedPass:   true,
			expectedPoolId: sdk.NewInt(2).Uint64(),
		},
		{
			name:         "Incorrect token denom, raise",
			firstToken:   "tokenA",
			secondToken:  "tokenC",
			expectedPass: false,
		},
		{
			name:         "Incorrect token denom, inverted",
			firstToken:   "tokenC",
			secondToken:  "tokenA",
			expectedPass: false,
		},
		{
			name:         "Incorrect token denom, same token",
			firstToken:   "tokenA",
			secondToken:  "tokenA",
			expectedPass: false,
		},
		{
			name:         "Incorrect token denom, missing token",
			firstToken:   "",
			secondToken:  "tokenA",
			expectedPass: false,
		},
		{
			name:         "Incorrect token denom, missing tokens",
			firstToken:   "",
			secondToken:  "",
			expectedPass: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			app.SpotKeeper.SetPool(ctx, types.Pool{
				Id: 1,
				PoolParams: types.PoolParams{
					SwapFee:  sdk.NewDecWithPrec(3, 2),
					ExitFee:  sdk.NewDecWithPrec(3, 2),
					PoolType: types.PoolType_BALANCER,
					A:        sdk.ZeroInt(),
				},
				PoolAssets: []types.PoolAsset{
					{
						Token:  sdk.NewCoin("tokenB", sdk.NewInt(1000)),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewCoin("tokenA", sdk.NewInt(1000)),
						Weight: sdk.NewInt(1),
					},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				Address:     "address1",
			})

			app.SpotKeeper.SetPool(ctx, types.Pool{
				Id: 2,
				PoolParams: types.PoolParams{
					SwapFee:  sdk.NewDecWithPrec(3, 2),
					ExitFee:  sdk.NewDecWithPrec(3, 2),
					PoolType: types.PoolType_BALANCER,
					A:        sdk.ZeroInt(),
				},
				PoolAssets: []types.PoolAsset{
					{
						Token:  sdk.NewCoin("tokenB", sdk.NewInt(1000)),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewCoin("tokenC", sdk.NewInt(1000)),
						Weight: sdk.NewInt(1),
					},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				Address:     "address2",
			})

			retrievedPool, err := app.SpotKeeper.FetchPoolFromPair(ctx, tc.firstToken, tc.secondToken)
			retrievedPoolId := retrievedPool.Id
			if tc.expectedPass {
				require.NoError(t, err)

				require.Equal(t, tc.expectedPoolId, retrievedPoolId)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestNewPool(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	poolCreationFeeCoin := sdk.NewInt64Coin(denoms.NIBI, 1000*common.Precision)
	app.SpotKeeper.SetParams(ctx, types.NewParams(
		/*startingPoolNumber=*/ 1,
		/*poolCreationFee=*/ sdk.NewCoins(poolCreationFeeCoin),
		/*whitelistedAssets*/ []string{
			"uatom",
			"uosmo",
		},
	))

	userAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	err := simapp.FundAccount(app.BankKeeper, ctx, userAddr, sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000)),
		poolCreationFeeCoin,
	))
	require.NoError(t, err)

	poolId, err := app.SpotKeeper.NewPool(ctx,
		// sender
		userAddr,
		// poolParams
		types.PoolParams{
			SwapFee:  sdk.NewDecWithPrec(3, 2),
			ExitFee:  sdk.NewDecWithPrec(3, 2),
			PoolType: types.PoolType_BALANCER,
			A:        sdk.ZeroInt(),
		},
		// poolAssets
		[]types.PoolAsset{
			{
				Token:  sdk.NewCoin("uatom", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
			{
				Token:  sdk.NewCoin("uosmo", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
		})
	require.NoError(t, err)

	retrievedPool, _ := app.SpotKeeper.FetchPool(ctx, poolId)

	require.Equal(t, types.Pool{
		Id:      1,
		Address: retrievedPool.Address, // address is random so can't test, just reuse value
		PoolParams: types.PoolParams{
			SwapFee:  sdk.NewDecWithPrec(3, 2),
			ExitFee:  sdk.NewDecWithPrec(3, 2),
			PoolType: types.PoolType_BALANCER,
			A:        sdk.ZeroInt(),
		},
		PoolAssets: []types.PoolAsset{
			{
				Token:  sdk.NewCoin("uatom", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1 << 30),
			},
			{
				Token:  sdk.NewCoin("uosmo", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1 << 30),
			},
		},
		TotalWeight: sdk.NewInt(2 << 30),
		TotalShares: sdk.NewCoin("nibiru/pool/1", sdk.NewIntWithDecimal(100, 6)),
	}, retrievedPool)
}

func TestNewPoolNotEnoughFunds(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	app.SpotKeeper.SetParams(ctx, types.NewParams(
		/*startingPoolNumber=*/ 1,
		/*poolCreationFee=*/ sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1000*common.Precision)),
		/*whitelistedAssets*/ []string{},
	))

	userAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	err := simapp.FundAccount(app.BankKeeper, ctx, userAddr, sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000)),
		sdk.NewCoin("unibi", sdk.NewInt(999*common.Precision)),
	))
	require.NoError(t, err)

	_, err = app.SpotKeeper.NewPool(ctx,
		// sender
		userAddr,
		// poolParams
		types.PoolParams{
			SwapFee:  sdk.NewDecWithPrec(3, 2),
			ExitFee:  sdk.NewDecWithPrec(3, 2),
			PoolType: types.PoolType_BALANCER,
		},
		// poolAssets
		[]types.PoolAsset{
			{
				Token:  sdk.NewCoin("uatom", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
			{
				Token:  sdk.NewCoin("uosmo", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
		})
	require.Error(t, err)
}

func TestNewPoolTooLittleAssets(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	userAddr, err := sdk.AccAddressFromBech32(testutil.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
		PoolType: types.PoolType_BALANCER,
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.SpotKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooFewPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestKeeperNewPoolNotWhitelistedAssets(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	userAddr, err := sdk.AccAddressFromBech32(testutil.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
		PoolType: types.PoolType_BALANCER,
	}

	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom1", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom2", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.SpotKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTokenNotAllowed)
	require.Equal(t, uint64(0), poolId)
}

func TestNewPoolTooManyAssets(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	userAddr, err := sdk.AccAddressFromBech32(testutil.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee:  sdk.NewDecWithPrec(3, 2),
		ExitFee:  sdk.NewDecWithPrec(3, 2),
		PoolType: types.PoolType_BALANCER,
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom1", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom2", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom3", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom4", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom5", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom6", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom7", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom8", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom9", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.SpotKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooManyPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestNewPoolDups(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	userAddr, err := sdk.AccAddressFromBech32(testutil.AccAddress().String())
	require.NoError(t, err)

	poolCreationFeeCoin := sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1))
	err = simapp.FundAccount(app.BankKeeper, ctx, userAddr, sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1000)),
		sdk.NewCoin("bar", sdk.NewInt(1000)),
		sdk.NewCoin("foo", sdk.NewInt(1000)),
	))
	require.NoError(t, err)
	app.SpotKeeper.SetParams(ctx, types.NewParams(
		/*startingPoolNumber=*/ 1,
		/*poolCreationFee=*/ poolCreationFeeCoin,
		/*whitelistedAssets*/ []string{
			"bar",
			"foo",
		},
	))

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []types.PoolAsset{
		{
			Token:  sdk.NewCoin("bar", sdk.NewInt(10)),
			Weight: sdk.NewInt(1),
		},
		{
			Token:  sdk.NewCoin("foo", sdk.NewInt(10)),
			Weight: sdk.NewInt(1),
		},
	}

	poolId, err := app.SpotKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.NoError(t, err)
	require.Equal(t, uint64(1), poolId)

	_, err = app.SpotKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrPoolWithSameAssetsExists)
}

func TestJoinPool(t *testing.T) {
	const shareDenom = "nibiru/pool/1"

	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPool              types.Pool
		tokensIn                 sdk.Coins
		expectedNumSharesOut     sdk.Coin
		expectedRemCoins         sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "join with all assets",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			expectedNumSharesOut:     sdk.NewInt64Coin(shareDenom, 100),
			expectedRemCoins:         sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(sdk.NewInt64Coin(shareDenom, 100)),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 200),
					sdk.NewInt64Coin("foo", 200),
				),
				/*shares=*/ 200),
		},
		{
			name: "join with some assets, none remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with some assets, some remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 75),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 25),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 150),
				),
				/*shares=*/ 150),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			joinerAddr := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds))

			pool, numSharesOut, remCoins, err := app.SpotKeeper.JoinPool(ctx, joinerAddr, 1, tc.tokensIn, false)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, pool)
			require.Equal(t, tc.expectedNumSharesOut, numSharesOut)
			require.Equal(t, tc.expectedRemCoins, remCoins)
		})
	}
}

func TestJoinPoolAllAssets(t *testing.T) {
	const shareDenom = "nibiru/pool/1"

	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPool              types.Pool
		tokensIn                 sdk.Coins
		expectedNumSharesOut     sdk.Coin
		expectedRemCoins         sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "join with all assets",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			expectedNumSharesOut:     sdk.NewInt64Coin(shareDenom, 100),
			expectedRemCoins:         sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(sdk.NewInt64Coin(shareDenom, 100)),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 200),
					sdk.NewInt64Coin("foo", 200),
				),
				/*shares=*/ 200),
		},
		{
			name: "join with some assets, none remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 50),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 150),
				),
				/*shares=*/ 150),
		},
		{
			name: "join with some assets, pool empty in one side, none remaining",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 1),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 904),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 904),
				sdk.NewInt64Coin("bar", 0),
				sdk.NewInt64Coin("foo", 0),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 101),
				),
				/*shares=*/ 1004),
		},
		{
			name: "join with some assets, but swap done",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100),
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 75),
			),
			expectedNumSharesOut: sdk.NewInt64Coin(shareDenom, 62),
			expectedRemCoins:     sdk.NewCoins(),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(shareDenom, 62),
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 25),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 150),
					sdk.NewInt64Coin("foo", 175),
				),
				/*shares=*/ 162),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			joinerAddr := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds))

			pool, numSharesOut, remCoins, err := app.SpotKeeper.JoinPool(ctx, joinerAddr, 1, tc.tokensIn, true)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, pool)
			require.Equal(t, tc.expectedNumSharesOut, numSharesOut)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, joinerAddr))
		})
	}
}

func TestExitPool(t *testing.T) {
	const shareDenom = "nibiru/pool/1"

	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPoolFunds         sdk.Coins
		initialPool              types.Pool
		poolSharesIn             sdk.Coin
		expectedTokensOut        sdk.Coins
		expectedJoinerFinalFunds sdk.Coins
		expectedFinalPool        types.Pool
	}{
		{
			name: "exit all pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			initialPoolFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 100),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 99),
				sdk.NewInt64Coin("foo", 99),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 199),
				sdk.NewInt64Coin("foo", 199),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1),
					sdk.NewInt64Coin("foo", 1),
				),
				/*shares=*/ 0,
			),
		},
		{
			name: "exit half pool shares",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
				sdk.NewInt64Coin(shareDenom, 100),
			),
			initialPoolFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesIn: sdk.NewInt64Coin(shareDenom, 50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 49),
				sdk.NewInt64Coin("foo", 49),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 149),
				sdk.NewInt64Coin("foo", 149),
				sdk.NewInt64Coin(shareDenom, 50),
			),
			expectedFinalPool: mock.SpotPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 51),
					sdk.NewInt64Coin("foo", 51),
				),
				/*shares=*/ 50,
			),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			poolAddr := testutil.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.SpotKeeper.SetPool(ctx, tc.initialPool)

			sender := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, sender, tc.joinerInitialFunds))
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, tc.initialPool.GetAddress(), tc.initialPoolFunds))

			tokensOut, err := app.SpotKeeper.ExitPool(ctx, sender, 1, tc.poolSharesIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokensOut, tokensOut)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, sender))
			pool, _ := app.SpotKeeper.FetchPool(ctx, 1)
			require.Equal(t, tc.expectedFinalPool, pool)
		})
	}
}
