package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestOpenPosition_Setup(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
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
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewInt(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)
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

				t.Log("Set vpool defined by pair on VpoolKeeper")
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
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewInt(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)

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

				t.Log("Set vpool defined by pair on VpoolKeeper")
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

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
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
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewInt(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)

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
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				marginDelta := sdk.OneDec()
				_, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, types.Position{
						Pair: "osmo:nusd",
					}, marginDelta)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "fail - invalid token pair passed to calculation",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				the3pool := "dai:usdc:usdt"
				marginDelta := sdk.OneDec()
				_, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, types.Position{Pair: the3pool}, marginDelta)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "signedRemainMargin negative bc of marginDelta",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 100%
				)
				premiumFractions := []sdk.Dec{sdk.ZeroDec()} // fPayment -> 0
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: premiumFractions,
				})

				pos := &types.Position{
					Address: alice.String(), Pair: pair.String(),
					Margin: sdk.NewDec(100), Size_: sdk.NewDec(200),
					LastUpdateCumulativePremiumFraction: premiumFractions[0],
				}

				marginDelta := sdk.NewDec(-300)
				remaining, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, *pos, marginDelta)
				require.NoError(t, err)
				// signedRemainMargin
				//   = marginDelta - fPayment + pos.Margin
				//   = -300 - 0 + 100 = -200
				// ∴ remaining.badDebt = signedRemainMargin.Abs() = 200
				require.EqualValues(t, sdk.NewDec(200), remaining.BadDebt)
				require.EqualValues(t, sdk.ZeroDec(), remaining.FPayment)
				require.EqualValues(t, sdk.Dec{}, remaining.Margin)
				require.EqualValues(t, sdk.ZeroDec(), remaining.LatestCPF)
			},
		},
		{
			name: "large fPayment lowers pos value by half",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 100%
				)
				premiumFractions := []sdk.Dec{
					sdk.MustNewDecFromStr("0.25"),
					sdk.MustNewDecFromStr("0.5"),
					sdk.MustNewDecFromStr("0.75"),
				}
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: premiumFractions,
				})

				pos := &types.Position{
					Address: alice.String(), Pair: pair.String(),
					Margin: sdk.NewDec(100), Size_: sdk.NewDec(200),
					LastUpdateCumulativePremiumFraction: premiumFractions[1],
				}

				marginDelta := sdk.NewDec(0)
				remaining, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, *pos, marginDelta)
				require.NoError(t, err)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.75"), remaining.LatestCPF)
				// FPayment
				//   = (remaining.LatestCPF - pos.LastUpdateCumulativePremiumFraction)
				//      * pos.Size_
				//   = (0.75 - 0.5) * 200
				//   = 50
				require.EqualValues(t, sdk.NewDec(50), remaining.FPayment)
				// signedRemainMargin
				//   = marginDelta - fPayment + pos.Margin
				//   = 0 - 50 + 100 = 50
				// ∴ remaining.BadDebt = 0
				// ∴ remaining.Margin = 50
				require.EqualValues(t, sdk.NewDec(0), remaining.BadDebt)
				require.EqualValues(t, sdk.NewDec(50), remaining.Margin)
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
		initialMargin  sdk.Dec
		addedMargin    sdk.Dec
		expectedMargin sdk.Dec
	}{
		{
			name:           "add margin",
			initialMargin:  sdk.NewDec(100),
			addedMargin:    sdk.NewDec(100),
			expectedMargin: sdk.NewDec(200),
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
					sdk.NewCoin(common.StableDenom, tc.addedMargin.TruncateInt()),
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
					Size_:   sdk.NewDec(9999),
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
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "negative margin remove - fail",
			test: func() {
				removeAmt := sdk.NewInt(-5)

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), Vpool: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "zero margin remove - fail",
			test: func() {
				removeAmt := sdk.ZeroInt()

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), Vpool: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "vpool doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), Vpool: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 100%
				)

				removeAmt := sdk.NewInt(5)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), Vpool: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := perpKeeper.RemoveMargin(
					goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "remove margin - happy path 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("xxx:yyy")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(10_000_000), //
					/* x */ sdk.NewDec(5_000_000), // 5 tokens
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 0.9 ratio
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
				})

				t.Log("Fund trader (Alice) account with sufficient quote")
				var err error
				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
				require.NoError(t, err)

				t.Log("Open long position with 5x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewInt(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)
				require.NoError(t, err)

				t.Log("Attempt to remove 10% of the position")
				removeAmt := sdk.NewInt(6)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), Vpool: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				// TODO: Blocker - Need GetOutputTWAP from prices.go
				// The test will panic b/c it's missing that implementation.
				require.Panics(t,
					func() {
						_, err := perpKeeper.RemoveMargin(goCtx, msg)
						require.Error(t, err)
					})
				// Desired behavior ↓
				// err = perpKeeper.RemoveMargin(
				// 	ctx, pair, alice, removeAmt)
				// require.NoError(t, err)
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
