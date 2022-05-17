package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestCreateLiquidation(t *testing.T) {
	testcases := []struct {
		name             string
		side             types.Side
		quote            sdk.Dec
		leverage         sdk.Dec
		baseLimit        sdk.Dec
		liquidationFee   sdk.Dec
		removeMargin     sdk.Dec
		startingQuote    sdk.Dec
		excpectedBadDebt sdk.Dec
		expectedPass     bool
	}{
		{
			name:             "happy path - Buy",
			side:             types.Side_BUY,
			quote:            sdk.MustNewDecFromStr("50"),
			leverage:         sdk.OneDec(),
			baseLimit:        sdk.ZeroDec(),
			liquidationFee:   sdk.MustNewDecFromStr("0.1"),
			startingQuote:    sdk.MustNewDecFromStr("60"),
			excpectedBadDebt: sdk.MustNewDecFromStr("0"),
			expectedPass:     true,
		},
		{
			name:             "happy path - Sell",
			side:             types.Side_BUY,
			quote:            sdk.MustNewDecFromStr("50"),
			leverage:         sdk.OneDec(),
			baseLimit:        sdk.ZeroDec(),
			liquidationFee:   sdk.MustNewDecFromStr("0.123123"),
			startingQuote:    sdk.MustNewDecFromStr("60"),
			excpectedBadDebt: sdk.MustNewDecFromStr("0"),
			expectedPass:     true,
		},
		{
			name:           "liquidateEmptyPositionBUY",
			side:           types.Side_BUY,
			quote:          sdk.MustNewDecFromStr("0"),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			startingQuote:  sdk.MustNewDecFromStr("60"),
			expectedPass:   false,
		},
		{
			name:           "liquidateEmptyPositionSELL",
			side:           types.Side_SELL,
			quote:          sdk.MustNewDecFromStr("0"),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			startingQuote:  sdk.MustNewDecFromStr("60"),
			expectedPass:   false,
		},
		{
			/* We open a position for 500k, with a liquidation fee of 50k.
			This means 25k for the liquidator, and 25k for the perp fund.
			Because the user only have margin for 50, we create 24950 of bad
			debt (2500 due to liquidator minus 50).
			*/
			name:             "happy path - BadDebt",
			side:             types.Side_SELL,
			quote:            sdk.MustNewDecFromStr("50"),
			leverage:         sdk.MustNewDecFromStr("10000"),
			baseLimit:        sdk.ZeroDec(),
			liquidationFee:   sdk.MustNewDecFromStr("0.1"),
			startingQuote:    sdk.MustNewDecFromStr("1150"),
			excpectedBadDebt: sdk.MustNewDecFromStr("24950"),
			expectedPass:     true,
		},
		{
			// Same as above case but for shorts
			name:             "happy path - BadDebt",
			side:             types.Side_BUY,
			quote:            sdk.MustNewDecFromStr("50"),
			leverage:         sdk.MustNewDecFromStr("10000"),
			baseLimit:        sdk.ZeroDec(),
			liquidationFee:   sdk.MustNewDecFromStr("0.1"),
			startingQuote:    sdk.MustNewDecFromStr("1150"),
			excpectedBadDebt: sdk.MustNewDecFromStr("24950"),
			expectedPass:     true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
				sdk.MustNewDecFromStr("1"),
				sdk.MustNewDecFromStr("0.1"),
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

			t.Log("Set vpool defined by pair on PerpKeeper")
			perpKeeper := &nibiruApp.PerpKeeper
			params := types.DefaultParams()

			perpKeeper.SetParams(ctx, types.NewParams(
				params.Stopped,
				params.MaintenanceMarginRatio,
				params.GetTollRatioAsDec(),
				params.GetSpreadRatioAsDec(),
				tc.liquidationFee,
				params.GetPartialLiquidationRatioAsDec(),
			))

			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       pair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Fund trader (Alice) account with sufficient quote")
			var err error
			alice := sample.AccAddress()
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
				sdk.NewCoins(sdk.NewInt64Coin("yyy", tc.startingQuote.TruncateInt64())))
			require.NoError(t, err)

			t.Log("Open position")
			err = nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.side, alice, tc.quote, tc.leverage, tc.baseLimit)

			require.NoError(t, err)

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
			if err != nil {
				panic(err)
			}

			t.Log("Liquidate the position")
			liquidationOutput, err := nibiruApp.PerpKeeper.CreateLiquidation(ctx, pair, alice, position)

			if tc.expectedPass {
				require.NoError(t, err)

				// We effectively closed the position
				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.Equal(t, sdk.ZeroDec(), newPosition.Size_)
				require.Equal(t, sdk.ZeroDec(), newPosition.Margin)
				require.Equal(t, sdk.ZeroDec(), newPosition.OpenNotional)

				// liquidator fee is half the liquidation fee of ExchangedQuoteAssetAmount
				require.Equal(t, liquidationOutput.PositionResp.ExchangedQuoteAssetAmount.Mul(tc.liquidationFee).Quo(sdk.MustNewDecFromStr("2")), liquidationOutput.FeeToLiquidator)
				require.Equal(t, tc.excpectedBadDebt, liquidationOutput.BadDebt)
			} else {
				require.Error(t, err)

				// No change in the position
				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.Equal(t, position.Size_, newPosition.Size_)
				require.Equal(t, position.Margin, newPosition.Margin)
				require.Equal(t, position.OpenNotional, newPosition.OpenNotional)
			}
		})
	}
}

func TestCreatePartialLiquidation(t *testing.T) {
	testcases := []struct {
		name                    string
		side                    types.Side
		quote                   sdk.Dec
		leverage                sdk.Dec
		baseLimit               sdk.Dec
		liquidationFee          sdk.Dec
		partialLiquidationRatio sdk.Dec
		removeMargin            sdk.Dec
		startingQuote           sdk.Dec
		expectedPass            bool

		excpectedBadDebt        sdk.Dec
		newPositionSize         sdk.Dec
		newPositionMargin       sdk.Dec
		newPositionOpenNotional sdk.Dec
		expectedFee             sdk.Dec
	}{

		{
			name:                    "happy path - Buy",
			side:                    types.Side_BUY,
			quote:                   sdk.MustNewDecFromStr("5000"),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.5"),
			startingQuote:           sdk.MustNewDecFromStr("6000"),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("1250"),
			// newPositionMargin = quote - expectedFee = 5000 - 250
			newPositionMargin:       sdk.MustNewDecFromStr("4750"),
			newPositionOpenNotional: sdk.MustNewDecFromStr("2500"),
			expectedFee:             sdk.MustNewDecFromStr("250"),
		},
		{
			name:                    "happy path - Sell",
			side:                    types.Side_SELL,
			quote:                   sdk.MustNewDecFromStr("5000"),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.5"),
			startingQuote:           sdk.MustNewDecFromStr("6000"),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("-1250"),
			// newPositionMargin = quote - expectedFee = 5000 - 250
			newPositionMargin:       sdk.MustNewDecFromStr("4750"),
			newPositionOpenNotional: sdk.MustNewDecFromStr("2500"),
			expectedFee:             sdk.MustNewDecFromStr("250"),
		},
		{
			name:                    "happy path - SellDifferentPercentage",
			side:                    types.Side_SELL,
			quote:                   sdk.MustNewDecFromStr("5000"),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.4"),
			startingQuote:           sdk.MustNewDecFromStr("6000"),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("-1500"),
			// newPositionMargin = quote - expectedFee = 5000 - 200
			newPositionMargin:       sdk.MustNewDecFromStr("4800"),
			newPositionOpenNotional: sdk.MustNewDecFromStr("3000"),
			expectedFee:             sdk.MustNewDecFromStr("200"),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)
			pair := common.TokenPair("xxx:yyy")

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				pair.String(),
				sdk.MustNewDecFromStr("0.9"),   // 0.9 ratio
				sdk.NewDec(10_000_000_000_000), //
				sdk.NewDec(5_000_000_000_000),  // 5 tokens
				sdk.MustNewDecFromStr("1"),
				sdk.MustNewDecFromStr("0.1"),
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

			t.Log("Set vpool defined by pair on PerpKeeper")
			perpKeeper := &nibiruApp.PerpKeeper
			params := types.DefaultParams()

			perpKeeper.SetParams(ctx, types.NewParams(
				params.Stopped,
				params.MaintenanceMarginRatio,
				params.GetTollRatioAsDec(),
				params.GetSpreadRatioAsDec(),
				tc.liquidationFee,
				tc.partialLiquidationRatio,
			))

			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       pair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Fund trader (Alice) account with sufficient quote")
			var err error
			alice := sample.AccAddress()
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
				sdk.NewCoins(sdk.NewInt64Coin("yyy", tc.startingQuote.TruncateInt64())))
			require.NoError(t, err)

			t.Log("Open position")
			err = nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.side, alice, tc.quote, tc.leverage, tc.baseLimit)

			require.NoError(t, err)

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
			if err != nil {
				panic(err)
			}

			t.Log("Liquidate the position")
			liquidationOutput, err := nibiruApp.PerpKeeper.CreatePartialLiquidation(
				ctx, pair, alice, position)

			if tc.expectedPass {
				require.NoError(t, err)

				t.Log("Verify successful setup of new position")
				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.InDelta(t,
					/* expected */ tc.newPositionSize.MustFloat64(),
					/* actual */ newPosition.Size_.MustFloat64(),
					/* delta */ 0.0001)
				require.InDelta(t,
					/* expected */ tc.newPositionMargin.MustFloat64(),
					/* actual */ newPosition.Margin.MustFloat64(),
					/* delta */ 0.0001)
				require.InDelta(t,
					/* expected */ tc.newPositionOpenNotional.MustFloat64(),
					/* actual */ newPosition.OpenNotional.MustFloat64(),
					/* delta */ 0.0001)

				t.Log("Verify correct values for liquidator fees and ExchangedQuoteAssetAmount")
				require.Equal(t,
					liquidationOutput.PositionResp.ExchangedQuoteAssetAmount.
						Mul(tc.liquidationFee).Quo(sdk.MustNewDecFromStr("2")),
					liquidationOutput.FeeToLiquidator)
				require.InDelta(t,
					/* expected */ tc.expectedFee.Quo(sdk.MustNewDecFromStr("2")).MustFloat64(),
					/* actual */ liquidationOutput.FeeToLiquidator.MustFloat64(),
					/* delta */ 0.0001)
				require.InDelta(t,
					/* expected */ tc.expectedFee.Quo(sdk.MustNewDecFromStr("2")).MustFloat64(),
					/* actual */ liquidationOutput.FeeToPerpEcosystemFund.MustFloat64(),
					/* delta */ 0.0001)
			} else {
				t.Log("Verify error raised and position didn't change")
				require.Error(t, err)

				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.Equal(t, position.Size_, newPosition.Size_)
				require.Equal(t, position.Margin, newPosition.Margin)
				require.Equal(t, position.OpenNotional, newPosition.OpenNotional)
			}
		})
	}
}
