package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/mock"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestGetAndSetNextPoolNumber(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	// Write to store
	app.DexKeeper.SetNextPoolNumber(ctx, 150)

	// Read from store
	poolNumber := app.DexKeeper.GetNextPoolNumber(ctx)

	require.EqualValues(t, poolNumber, 150)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	// Write a pool number
	app.DexKeeper.SetNextPoolNumber(ctx, 200)

	// Get next and increment should return the current pool number
	poolNumber := app.DexKeeper.GetNextPoolNumberAndIncrement(ctx)
	require.EqualValues(t, poolNumber, 200)

	// Check that the previous call incremented the number
	poolNumber = app.DexKeeper.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 201)
}

func TestSetAndFetchPool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	pool := types.Pool{
		Id: 150,
		PoolParams: types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
		},
		PoolAssets: []types.PoolAsset{
			types.PoolAsset{
				Token:  sdk.NewCoin("validatortoken", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
			types.PoolAsset{
				Token:  sdk.NewCoin("stake", sdk.NewInt(1000)),
				Weight: sdk.NewInt(1),
			},
		},
		TotalWeight: sdk.NewInt(2),
		TotalShares: sdk.NewInt64Coin("matrix/pool/150", 100),
	}

	app.DexKeeper.SetPool(ctx, pool)

	retrievedPool := app.DexKeeper.FetchPool(ctx, 150)

	require.Equal(t, pool, retrievedPool)
}

func TestNewPool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	app.DexKeeper.SetParams(ctx, types.NewParams(
		/*startingPoolNumber=*/ 1,
		/*poolCreationFee=*/ sdk.NewCoins(sdk.NewInt64Coin(common.GovDenom, 1000_000_000)),
	))

	userAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	err := simapp.FundAccount(app.BankKeeper, ctx, userAddr, sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000)),
		sdk.NewCoin("umtrx", sdk.NewInt(1000_000_000)),
	))
	require.NoError(t, err)

	poolId, err := app.DexKeeper.NewPool(ctx,
		// sender
		userAddr,
		// poolParams
		types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
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

	retrievedPool := app.DexKeeper.FetchPool(ctx, poolId)

	require.Equal(t, types.Pool{
		Id:      1,
		Address: retrievedPool.Address, // address is random so can't test, just reuse value
		PoolParams: types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
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
		TotalShares: sdk.NewCoin("matrix/pool/1", sdk.NewIntWithDecimal(100, 18)),
	}, retrievedPool)
}

func TestNewPoolNotEnoughFunds(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)

	app.DexKeeper.SetParams(ctx, types.NewParams(
		/*startingPoolNumber=*/ 1,
		/*poolCreationFee=*/ sdk.NewCoins(sdk.NewInt64Coin(common.GovDenom, 1000_000_000)),
	))

	userAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	err := simapp.FundAccount(app.BankKeeper, ctx, userAddr, sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000)),
		sdk.NewCoin("umtrx", sdk.NewInt(999_000_000)),
	))
	require.NoError(t, err)

	_, err = app.DexKeeper.NewPool(ctx,
		// sender
		userAddr,
		// poolParams
		types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
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
	app, ctx := testutil.NewMatrixApp(true)
	userAddr, err := sdk.AccAddressFromBech32(sample.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.DexKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooFewPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestNewPoolTooManyAssets(t *testing.T) {
	app, ctx := testutil.NewMatrixApp(true)
	userAddr, err := sdk.AccAddressFromBech32(sample.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
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

	poolId, err := app.DexKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooManyPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestJoinPool(t *testing.T) {
	const shareDenom = "matrix/pool/1"

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
			initialPool: mock.DexPool(
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
			expectedFinalPool: mock.DexPool(
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
			initialPool: mock.DexPool(
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
			expectedFinalPool: mock.DexPool(
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
			initialPool: mock.DexPool(
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
			expectedFinalPool: mock.DexPool(
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
			app, ctx := testutil.NewMatrixApp(true)

			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.DexKeeper.SetPool(ctx, tc.initialPool)

			joinerAddr := sample.AccAddress()
			simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds)

			pool, numSharesOut, remCoins, err := app.DexKeeper.JoinPool(ctx, joinerAddr, 1, tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, pool)
			require.Equal(t, tc.expectedNumSharesOut, numSharesOut)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, joinerAddr))
		})
	}
}

func TestExitPool(t *testing.T) {
	const shareDenom = "matrix/pool/1"

	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPoolFunds         sdk.Coins
		initialPool              types.Pool
		poolSharesOut            sdk.Coin
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
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesOut: sdk.NewInt64Coin(shareDenom, 100),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 99),
				sdk.NewInt64Coin("foo", 99),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 199),
				sdk.NewInt64Coin("foo", 199),
			),
			expectedFinalPool: mock.DexPool(
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
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 100),
					sdk.NewInt64Coin("foo", 100),
				),
				/*shares=*/ 100,
			),
			poolSharesOut: sdk.NewInt64Coin(shareDenom, 50),
			expectedTokensOut: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 49),
				sdk.NewInt64Coin("foo", 49),
			),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 149),
				sdk.NewInt64Coin("foo", 149),
				sdk.NewInt64Coin(shareDenom, 50),
			),
			expectedFinalPool: mock.DexPool(
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
			app, ctx := testutil.NewMatrixApp(true)

			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.DexKeeper.SetPool(ctx, tc.initialPool)

			sender := sample.AccAddress()
			simapp.FundAccount(app.BankKeeper, ctx, sender, tc.joinerInitialFunds)
			simapp.FundAccount(app.BankKeeper, ctx, tc.initialPool.GetAddress(), tc.initialPoolFunds)

			tokensOut, err := app.DexKeeper.ExitPool(ctx, sender, 1, tc.poolSharesOut)
			require.NoError(t, err)
			require.Equal(t, tc.expectedTokensOut, tokensOut)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, sender))
			require.Equal(t, tc.expectedFinalPool, app.DexKeeper.FetchPool(ctx, 1))
		})
	}
}
