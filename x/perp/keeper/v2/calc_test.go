package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestCalcFreeCollateralSuccess(t *testing.T) {
	testCases := []struct {
		name string

		positionSize           sdk.Dec
		marketDirection        v2types.Direction
		positionNotional       sdk.Dec
		expectedFreeCollateral sdk.Dec
	}{
		{
			name:                   "long position, zero PnL",
			positionSize:           sdk.OneDec(),
			marketDirection:        v2types.Direction_LONG,
			positionNotional:       sdk.NewDec(1000),
			expectedFreeCollateral: sdk.MustNewDecFromStr("37.5"),
		},
		{
			name:                   "long position, positive PnL",
			positionSize:           sdk.OneDec(),
			marketDirection:        v2types.Direction_LONG,
			positionNotional:       sdk.NewDec(1100),
			expectedFreeCollateral: sdk.MustNewDecFromStr("31.25"),
		},
		{
			name:                   "long position, negative PnL",
			marketDirection:        v2types.Direction_LONG,
			positionSize:           sdk.OneDec(),
			positionNotional:       sdk.NewDec(970),
			expectedFreeCollateral: sdk.MustNewDecFromStr("9.375"),
		},
		{
			name:                   "long position, huge negative PnL",
			marketDirection:        v2types.Direction_LONG,
			positionSize:           sdk.OneDec(),
			positionNotional:       sdk.NewDec(900),
			expectedFreeCollateral: sdk.MustNewDecFromStr("-56.25"),
		},
		{
			name:                   "short position, zero PnL",
			positionSize:           sdk.OneDec().Neg(),
			marketDirection:        v2types.Direction_SHORT,
			positionNotional:       sdk.NewDec(1000),
			expectedFreeCollateral: sdk.MustNewDecFromStr("37.5"),
		},
		{
			name:                   "short position, positive PnL",
			positionSize:           sdk.OneDec().Neg(),
			marketDirection:        v2types.Direction_SHORT,
			positionNotional:       sdk.NewDec(900),
			expectedFreeCollateral: sdk.MustNewDecFromStr("43.75"),
		},
		{
			name:                   "short position, negative PnL",
			positionSize:           sdk.OneDec().Neg(),
			marketDirection:        v2types.Direction_SHORT,
			positionNotional:       sdk.NewDec(1030),
			expectedFreeCollateral: sdk.MustNewDecFromStr("5.625"),
		},
		{
			name:                   "short position, huge negative PnL",
			positionSize:           sdk.OneDec().Neg(),
			marketDirection:        v2types.Direction_SHORT,
			positionNotional:       sdk.NewDec(1100),
			expectedFreeCollateral: sdk.MustNewDecFromStr("-68.75"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			k, _, ctx := getKeeper(t)

			market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			amm := v2types.AMM{}
			pos := v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           tc.positionSize,
				Margin:                          sdk.NewDec(100),
				OpenNotional:                    sdk.NewDec(1000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			}

			freeCollateral, err := k.calcFreeCollateral(ctx, market, amm, pos)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedFreeCollateral, freeCollateral)
		})
	}
}

func TestCalcRemainMarginWithFundingPayment(t *testing.T) {
	testCases := []struct {
		name            string
		position        v2types.Position
		marginDelta     sdk.Dec
		marketLatestCPF sdk.Dec

		expectedMargin         sdk.Dec
		expectedBadDebt        sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name: "signedRemainMargin negative bc of marginDelta",
			position: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(1),
				Margin:                          sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			marginDelta:     sdk.NewDec(-300),
			marketLatestCPF: sdk.ZeroDec(),

			expectedMargin:         sdk.ZeroDec(),
			expectedBadDebt:        sdk.NewDec(200),
			expectedFundingPayment: sdk.ZeroDec(),
		},
		{
			name: "large fPayment lowers pos value by half",
			position: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(200),
				Margin:                          sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			marginDelta:     sdk.ZeroDec(),
			marketLatestCPF: sdk.MustNewDecFromStr("0.25"),

			expectedMargin:         sdk.NewDec(50),
			expectedBadDebt:        sdk.ZeroDec(),
			expectedFundingPayment: sdk.NewDec(50),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			remaining := CalcRemainMarginWithFundingPayment(tc.position, tc.marginDelta, tc.marketLatestCPF)
			assert.EqualValues(t, tc.expectedMargin, remaining.MarginAbs)
			assert.EqualValues(t, tc.expectedBadDebt, remaining.BadDebtAbs)
			assert.EqualValues(t, tc.expectedFundingPayment, remaining.FundingPayment)
		})
	}
}
