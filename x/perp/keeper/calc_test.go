package keeper_test

import (
	"testing"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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
				require.ErrorIs(t, err, collections.ErrNotFound)
			},
		},
		{
			name: "signedRemainMargin negative bc of marginDelta",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
				trader := testutil.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* y */ sdk.NewDec(1*common.Precision), //
					/* x */ sdk.NewDec(1*common.Precision), //
					vpooltypes.VpoolConfig{
						FluctuationLimitRatio:  sdk.MustNewDecFromStr("1.0"),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
						MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("1.0"), // 100%,
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					},
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})

				pos := &types.Position{
					TraderAddress:                   trader.String(),
					Pair:                            pair,
					Margin:                          sdk.NewDec(100),
					Size_:                           sdk.NewDec(200),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
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
				trader := testutil.AccAddress()
				pair := common.MustNewAssetPair("osmo:nusd")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					/* y */ sdk.NewDec(1*common.Precision), //
					/* x */ sdk.NewDec(1*common.Precision), //
					vpooltypes.VpoolConfig{
						FluctuationLimitRatio:  sdk.MustNewDecFromStr("1.0"),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
						MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("1.0"), // 100%,
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					},
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.75"),
				})

				pos := &types.Position{
					TraderAddress:                   trader.String(),
					Pair:                            pair,
					Margin:                          sdk.NewDec(100),
					Size_:                           sdk.NewDec(200),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.5"),
				}

				marginDelta := sdk.NewDec(0)
				remaining, err := nibiruApp.PerpKeeper.CalcRemainMarginWithFundingPayment(
					ctx, *pos, marginDelta)
				require.NoError(t, err)
				require.EqualValues(t, sdk.MustNewDecFromStr("0.75"), remaining.LatestCumulativePremiumFraction)
				// FPayment
				//   = (remaining.LatestCPF - pos.LatestCumulativePremiumFraction)
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
