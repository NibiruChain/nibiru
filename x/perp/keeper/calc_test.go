package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

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
				require.EqualValues(t, sdk.ZeroDec(), remaining.FundingPayment)
				require.EqualValues(t, sdk.ZeroDec(), remaining.Margin)
				require.EqualValues(t, sdk.ZeroDec(), remaining.LatestCumulativePremiumFraction)
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
				require.EqualValues(t, sdk.MustNewDecFromStr("0.75"), remaining.LatestCumulativePremiumFraction)
				// FPayment
				//   = (remaining.LatestCPF - pos.LastUpdateCumulativePremiumFraction)
				//      * pos.Size_
				//   = (0.75 - 0.5) * 200
				//   = 50
				require.EqualValues(t, sdk.NewDec(50), remaining.FundingPayment)
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
