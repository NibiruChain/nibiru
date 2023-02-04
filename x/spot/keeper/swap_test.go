package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/spot/types"
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
			name: "testnet 2 BUG, should not panic",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 236534500),
				sdk.NewInt64Coin("unusd", 1700000000),
				sdk.NewInt64Coin("uusdt", 701785070),
			),
			initialPool: types.Pool{
				Id: 1,
				PoolParams: types.PoolParams{
					SwapFee:  sdk.MustNewDecFromStr("0.01"),
					ExitFee:  sdk.MustNewDecFromStr("0.01"),
					PoolType: types.PoolType_STABLESWAP,
					A:        sdk.NewInt(10),
				},
				PoolAssets: []types.PoolAsset{
					{Token: sdk.NewInt64Coin("unusd", 1_510_778_598),
						Weight: sdk.NewInt(1)},
					{Token: sdk.NewInt64Coin("uusdt", 7_712_056),
						Weight: sdk.NewInt(1)},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			tokenIn:          sdk.NewInt64Coin("unusd", 1_500_000_000),
			tokenOutDenom:    "uusdt",
			expectedTokenOut: sdk.NewInt64Coin("uusdt", 6_670_336),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 236_534_500),
				sdk.NewInt64Coin("unusd", 200_000_000),
				sdk.NewInt64Coin("uusdt", 708_455_406),
			),
			expectedFinalPool: types.Pool{
				Id: 1,
				PoolParams: types.PoolParams{
					SwapFee:  sdk.MustNewDecFromStr("0.01"),
					ExitFee:  sdk.MustNewDecFromStr("0.01"),
					PoolType: types.PoolType_STABLESWAP,
					A:        sdk.NewInt(10),
				},
				PoolAssets: []types.PoolAsset{
					{Token: sdk.NewInt64Coin("unusd", 3_010_778_598),
						Weight: sdk.NewInt(1)},
					{Token: sdk.NewInt64Coin("uusdt", 1_041_720),
						Weight: sdk.NewInt(1)},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			expectedError: nil,
		},
		{
			name: "regular stableswap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.USDC, 10),
			),
			initialPool: mock.DexStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.USDC, 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:          sdk.NewInt64Coin(denoms.USDC, 10),
			tokenOutDenom:    denoms.NUSD,
			expectedTokenOut: sdk.NewInt64Coin(denoms.NUSD, 10),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NUSD, 10),
			),
			expectedFinalPool: mock.DexStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.USDC, 110),
					sdk.NewInt64Coin(denoms.NUSD, 90),
				),
				/*shares=*/ 100,
			),
			expectedError: nil,
		},
		{
			name: "regular swap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 100),
			),
			initialPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:          sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom:    denoms.NUSD,
			expectedTokenOut: sdk.NewInt64Coin(denoms.NUSD, 50),
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.NUSD, 50),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 200),
					sdk.NewInt64Coin(denoms.NUSD, 50),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("unibi", 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("unibi", 1),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			tokenIn:       sdk.NewInt64Coin("foo", 100),
			tokenOutDenom: denoms.NUSD,
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin("foo", 100),
			),
			expectedFinalPool: mock.DexPool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin("unibi", 100),
					sdk.NewInt64Coin(denoms.NUSD, 100),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
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
					sdk.NewInt64Coin(denoms.NUSD, 100),
				),
				/*shares=*/ 100,
			),
			expectedError: types.ErrSameTokenDenom,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewTestNibiruAppAndContext(true)

			// fund pool account
			poolAddr := testutil.AccAddress()
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
			sender := testutil.AccAddress()
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

func TestDoubleSwapExactAmountIn(t *testing.T) {
	tests := []struct {
		name string

		// test setup
		userInitialFunds sdk.Coins
		initialPool      types.Pool
		tokenIns         []sdk.Coin
		tokenOutDenoms   []string

		// expected results
		expectedTokenOuts      []sdk.Coin
		expectedUserFinalFunds sdk.Coins
		expectedFinalPool      types.Pool
	}{
		{
			name: "double stableswap",
			userInitialFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.USDC, 10_000),
			),
			initialPool: mock.DexStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.USDC, 100_000_000),
					sdk.NewInt64Coin(denoms.NUSD, 100_000_000),
				),
				/*shares=*/ 100,
			),
			tokenIns:          []sdk.Coin{sdk.NewInt64Coin(denoms.USDC, 10_000), sdk.NewInt64Coin("unusd", 10_000)},
			tokenOutDenoms:    []string{denoms.NUSD, denoms.USDC},
			expectedTokenOuts: []sdk.Coin{sdk.NewInt64Coin(denoms.NUSD, 10_000), sdk.NewInt64Coin(denoms.USDC, 10_001)},
			expectedUserFinalFunds: sdk.NewCoins(
				sdk.NewInt64Coin(denoms.USDC, 10_001), // TODO: fix https://github.com/NibiruChain/nibiru/issues/1152
			),
			expectedFinalPool: mock.DexStablePool(
				/*poolId=*/ 1,
				/*assets=*/ sdk.NewCoins(
					sdk.NewInt64Coin(denoms.USDC, 99_999_999),
					sdk.NewInt64Coin(denoms.NUSD, 100_000_000),
				),
				/*shares=*/ 100,
			),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewTestNibiruAppAndContext(true)

			// fund pool account
			poolAddr := testutil.AccAddress()
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
			sender := testutil.AccAddress()
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, sender, tc.userInitialFunds))

			// swap assets
			for i, tokenIn := range tc.tokenIns {
				tokenOut, err := app.DexKeeper.SwapExactAmountIn(ctx, sender, tc.initialPool.Id, tokenIn, tc.tokenOutDenoms[i])
				require.NoError(t, err)

				fmt.Println("-------------", i)
				finalPool, err := app.DexKeeper.FetchPool(ctx, tc.initialPool.Id)
				require.NoError(t, err)
				fmt.Println(finalPool.PoolAssets)

				require.NoError(t, err)
				require.Equal(t, tc.expectedTokenOuts[i], tokenOut)
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
