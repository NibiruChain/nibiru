package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestOpenPosition_Setup(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "uninitialized vpool has no metadata | GetLatestCumulativePremiumFraction",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				vpool := common.TokenPair("xxx:yyy")
				lcpf, err := nibiruApp.PerpKeeper.GetLatestCumulativePremiumFraction(
					ctx, vpool)
				require.Error(t, err)
				require.EqualValues(t, sdk.Int{}, lcpf)
			},
		},
		{
			name: "open pos - uninitialized pool raised pair not supported error",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader without a vpool.")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")
				alice := sample.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewInt(10)
				baseLimit := sdk.NewInt(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice.String(), quote, leverage, baseLimit)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - vpool not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				)

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				alice := sample.AccAddress()
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewInt(10)
				baseLimit := sdk.NewInt(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice.String(), quote, leverage, baseLimit)

				fmt.Println(err.Error())
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - happy path 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				)

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: []sdk.Int{sdk.OneInt()},
				})

				t.Log("Fund trader (Alice) account with sufficient quote")
				var err error
				alice := sample.AccAddress()
				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
				require.NoError(t, err)

				t.Log("Open long position with 10x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewInt(10)
				baseLimit := sdk.NewInt(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice.String(), quote, leverage, baseLimit)

				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestCalcRemainMarginWithFundingPayment(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "get - no positions set raises vpool not found error",
			test: func() {
				vpool := common.TokenPair("osmo:nusd")

				nibiruApp, ctx := testutil.NewNibiruApp(true)

				marginDelta := sdk.OneInt()
				_, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, vpool, &types.Position{}, marginDelta)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestAddMargin(t *testing.T) {
	tests := []struct {
		name           string
		initialMargin  sdk.Int
		addedMargin    sdk.Int
		expectedMargin sdk.Int
	}{
		{
			name:           "add margin",
			initialMargin:  sdk.NewIntFromUint64(100),
			addedMargin:    sdk.NewIntFromUint64(100),
			expectedMargin: sdk.NewIntFromUint64(200),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testutil.NewNibiruApp(true)

			tokenPair, err := common.NewTokenPairFromStr("atom:nusd")
			require.NoError(t, err)

			t.Log("add margin funds (NUSD) to trader's account")
			traderAddr := sample.AccAddress()
			err = simapp.FundAccount(
				app.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(common.StableDenom, tc.addedMargin),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("establish initial position")
			app.PerpKeeper.SetPosition(
				ctx,
				tokenPair,
				traderAddr.String(),
				&types.Position{
					Address: traderAddr.String(),
					Pair:    tokenPair.String(),
					Size_:   sdk.NewIntFromUint64(9999),
					Margin:  tc.initialMargin,
				},
			)

			require.NoError(t,
				app.PerpKeeper.AddMargin(ctx, tokenPair, traderAddr, tc.addedMargin))

			position, err := app.PerpKeeper.GetPosition(
				ctx, tokenPair, traderAddr.String())
			require.NoError(t, err)
			require.Equal(t, tc.expectedMargin, position.Margin)
		})
	}
}

func TestRemoveMargin(t *testing.T) {

}
