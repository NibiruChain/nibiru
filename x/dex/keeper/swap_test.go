package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/dex/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestSwapExactAmountIn(t *testing.T) {
	tests := []struct {
		name string

		// test setup
		userInitialFunds sdk.Coins
		initialPool      types.Pool
		tokenIn          sdk.Coin
		tokenOutDenom    string

		// expected results
		expectedError          error
		expectedTokenOut       sdk.Coin
		expectedUserFinalFunds sdk.Coins
		expectedFinalPool      types.Pool
	}{
		{
			name: "regular swap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:          sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom:    "unusd",
			expectedTokenOut: sdk.NewInt64Coin("unusd", 50),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unusd", 50),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 200),
					sdk.NewInt64Coin("unusd", 50),
				),
				/*shares=*/ 100,
			),
			expectedError: nil,
		},
		{
			name: "not enough user funds",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 1),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom: "unusd",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 1),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			expectedError: sdkerrors.ErrInsufficientFunds,
		},
		{
			name: "invalid token in denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("foo", 100),
			tokenOutDenom: "unusd",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "invalid token out denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom: "foo",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrTokenDenomNotFound,
		},
		{
			name: "same token in and token out denom",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom: "unibi",
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin("unusd", 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrSameTokenDenom,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)

			// fund pool account
			poolAddr := sample.AccAddress()
			tc.initialPool.Address = poolAddr.String()
			tc.expectedFinalPool.Address = poolAddr.String()
			require.NoError(t,
				simapp.FundAccount(
					app.BankKeeper,
					ctx,
					poolAddr,
					tc.initialPool.PoolBalances(),
				),
			)
			app.DexKeeper.SetPool(ctx, tc.initialPool)

			// fund user account
			sender := sample.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, sender, tc.userInitialFunds))

			// swap assets
			tokenOut, err := app.DexKeeper.SwapExactAmountIn(ctx, sender, tc.initialPool.Id, tc.tokenIn, tc.tokenOutDenom)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedTokenOut, tokenOut)
			}

			// check user's final funds
			require.Equal(t,
				tc.expectedUserFinalFunds,
				app.BankKeeper.GetAllBalances(ctx, sender),
			)

			// check final pool state
			finalPool, err := app.DexKeeper.FetchPool(ctx, tc.initialPool.Id)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalPool, finalPool)
		})
	}
}
