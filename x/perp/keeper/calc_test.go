package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
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
				nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)

				marginDelta := sdk.OneDec()
				_, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, types.Position{
						Pair: common.Pair_NIBI_NUSD,
					}, marginDelta)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairMetadataNotFound.Error())
			},
		},
		{
			name: "signedRemainMargin negative bc of marginDelta",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctuationLimit */ sdk.MustNewDecFromStr("1.0"), // 100%
					/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("1.0"), // 100%
					/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
					/* maxLeverage */ sdk.MustNewDecFromStr("15"),
				)
				premiumFractions := []sdk.Dec{sdk.ZeroDec()} // fPayment -> 0
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
					Pair:                   pair,
					CumulativeFundingRates: premiumFractions,
				})

				pos := &types.Position{
					TraderAddress:                  trader.String(),
					Pair:                           pair,
					Margin:                         sdk.NewDec(100),
					Size_:                          sdk.NewDec(200),
					LatestCumulativeFundingPayment: premiumFractions[0],
				}

				marginDelta := sdk.NewDec(-300)
				remaining, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, *pos, marginDelta)
				require.NoError(t, err)
				// signedRemainMargin
				//   = marginDelta - fPayment + pos.Margin
				//   = -300 - 0 + 100 = -200
				// ∴ remaining.badDebt = signedRemainMargin.Abs() = 200
				require.True(t, sdk.NewDec(200).Equal(remaining.BadDebt))
				require.True(t, sdk.NewDec(0).Equal(remaining.FundingPayment))
				require.True(t, sdk.NewDec(0).Equal(remaining.Margin))
				require.EqualValues(t, sdk.ZeroDec(), remaining.LatestCumulativePremiumFraction)
			},
		},
		{
			name: "large fPayment lowers pos value by half",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
				trader := sample.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctuationLimit */ sdk.MustNewDecFromStr("1.0"), // 100%
					/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("1.0"), // 100%
					/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
					/* maxLeverage */ sdk.MustNewDecFromStr("15"),
				)
				premiumFractions := []sdk.Dec{
					sdk.MustNewDecFromStr("0.25"),
					sdk.MustNewDecFromStr("0.5"),
					sdk.MustNewDecFromStr("0.75"),
				}
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
					Pair:                   pair,
					CumulativeFundingRates: premiumFractions,
				})

				pos := &types.Position{
					TraderAddress:                  trader.String(),
					Pair:                           pair,
					Margin:                         sdk.NewDec(100),
					Size_:                          sdk.NewDec(200),
					LatestCumulativeFundingPayment: premiumFractions[1],
				}

				marginDelta := sdk.NewDec(0)
				remaining, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, *pos, marginDelta)
				require.NoError(t, err)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.75"), remaining.LatestCumulativePremiumFraction)
				// FPayment
				//   = (remaining.LatestCPF - pos.LatestCumulativeFundingPayment)
				//      * pos.Size_
				//   = (0.75 - 0.5) * 200
				//   = 50
				require.True(t, sdk.NewDec(50).Equal(remaining.FundingPayment))
				// signedRemainMargin
				//   = marginDelta - fPayment + pos.Margin
				//   = 0 - 50 + 100 = 50
				// ∴ remaining.BadDebt = 0
				// ∴ remaining.Margin = 50
				require.True(t, sdk.NewDec(0).Equal(remaining.BadDebt))
				require.True(t, sdk.NewDec(50).Equal(remaining.Margin))
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
