package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/mock"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSwapExactAmountIn(t *testing.T) {

	tests := []struct {
		name                     string
		joinerInitialFunds       sdk.Coins
		initialPool              types.Pool
		tokenIn                  sdk.Coin
		tokenOutDenom            string
		expectedTokenOut         sdk.Coin
		expectedFinalPool        types.Pool
		expectedJoinerFinalFunds sdk.Coins
	}{
		{
			name: "join with all of user's assets",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1_000_000),
					sdk.NewInt64Coin("foo", 1_000_000),
				),
				/*shares=*/ 100),
			tokenIn:          sdk.NewInt64Coin("bar", 100),
			tokenOutDenom:    "foo",
			expectedTokenOut: sdk.NewInt64Coin("foo", 99),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 99),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1_000_100),
					sdk.NewInt64Coin("foo", 999_901),
				),
				/*shares=*/ 100),
		},
		{
			name: "join with some of user's assets",
			joinerInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 100),
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1_000_000),
					sdk.NewInt64Coin("foo", 1_000_000),
				),
				/*shares=*/ 100),
			tokenIn:          sdk.NewInt64Coin("bar", 50),
			tokenOutDenom:    "foo",
			expectedTokenOut: sdk.NewInt64Coin("foo", 49),
			expectedJoinerFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("bar", 50),
				sdk.NewInt64Coin("foo", 149),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("bar", 1_000_050),
					sdk.NewInt64Coin("foo", 999_951),
				),
				/*shares=*/ 100),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewMatrixApp(true)

			// set up pool address and funds
			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			app.DexKeeper.SetPool(ctx, tc.initialPool)
			simapp.FundAccount(
				app.BankKeeper,
				ctx,
				poolAddr,
				tc.initialPool.PoolBalances(),
			)

			// set up user's funds
			joinerAddr := sample.AccAddress()
			simapp.FundAccount(app.BankKeeper, ctx, joinerAddr, tc.joinerInitialFunds)

			tokenOut, err := app.DexKeeper.SwapExactAmountIn(ctx, joinerAddr, 1, tc.tokenIn, tc.tokenOutDenom)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, app.DexKeeper.FetchPool(ctx, 1))
			require.Equal(t, tc.expectedTokenOut, tokenOut)
			require.Equal(t, tc.expectedJoinerFinalFunds, app.BankKeeper.GetAllBalances(ctx, joinerAddr))
		})
	}
}
