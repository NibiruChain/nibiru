package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeper "github.com/NibiruChain/nibiru/x/perp/keeper"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestCreateLiquidation(t *testing.T) {
	testcases := []struct {
		name              string
		side              types.Side
		quote             sdk.Dec
		leverage          sdk.Dec
		baseLimit         sdk.Dec
		liquidationFee    sdk.Dec
		removeMargin      sdk.Dec
		liquidationOutput keeper.LiquidationOutput
		startingQuote     sdk.Dec
		excpectedBadDebt  sdk.Dec
		expectedPass      bool
	}{
		{
			name:             "happPathBuy",
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
			name:             "happPathSell",
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
			/*
				We open a position for 500k, with a liquidation fee of 50k.
				This means 25k for the liquidator, and 25k for the perp fund.
				Because the user only have margin for 50, we create 24950 of bad debt (2500 due to liquidator minus 50).
			*/
			name:             "happPathBadDebt",
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
			/*
				Same but for shorts
			*/
			name:             "happPathBadDebt",
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
			fmt.Println(position)

			fmt.Println("FeeToInsuranceFund", liquidationOutput.FeeToInsuranceFund)
			fmt.Println("FeeToLiquidator", liquidationOutput.FeeToLiquidator)
			fmt.Println("BadDebt", liquidationOutput.BadDebt)
			//fmt.Println("PositionResp", liquidationOutput.PositionResp)

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
