package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestCreateLiquidation(t *testing.T) {
	testcases := []struct {
		name                    string
		side                    types.Side
		quote                   sdk.Dec
		leverage                sdk.Dec
		baseLimit               sdk.Dec
		liquidationFee          sdk.Dec
		removeMargin            sdk.Dec
		startingQuote           sdk.Dec
		expectedFeeToLiquidator sdk.Coin
		expectedPerpEFBalance   sdk.Coin
		excpectedBadDebt        sdk.Dec
		expectedPass            bool
	}{
		{
			name:                    "happy path - Buy",
			side:                    types.Side_BUY,
			quote:                   sdk.MustNewDecFromStr("50"),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			startingQuote:           sdk.MustNewDecFromStr("60"),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 2),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 1_000_045),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,
		},
		{
			name:                    "happy path - Sell",
			side:                    types.Side_BUY,
			quote:                   sdk.MustNewDecFromStr("50"),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.123123"),
			startingQuote:           sdk.MustNewDecFromStr("60"),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 3),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 1_000_043),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,
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
			name:           "happy path - BadDebt, long",
			side:           types.Side_SELL,
			quote:          sdk.MustNewDecFromStr("50"),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			startingQuote:  sdk.MustNewDecFromStr("1150"),
			// liquidationAmount = quote * leverage = 50 * 10_000 = 50_000
			// feeToLiquidator = liquidationAmount / 2 = 25_000
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			// perpEFBalance = startBalance - feeToLiquidator + quote
			//   = 1_000_000 - 25_000 + 50 = 975_550
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 975_550),
			excpectedBadDebt:      sdk.MustNewDecFromStr("24950"),
			expectedPass:          true,
		},
		{
			// Same as above case but for shorts
			name:                    "happy path - BadDebt, short",
			side:                    types.Side_BUY,
			quote:                   sdk.MustNewDecFromStr("50"),
			leverage:                sdk.MustNewDecFromStr("10000"),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			startingQuote:           sdk.MustNewDecFromStr("1150"),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 975_550),
			excpectedBadDebt:        sdk.MustNewDecFromStr("24950"),
			expectedPass:            true,
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
			require.NoError(t, err)

			t.Log("Artificially populate Vault and PerpEF to prevent BankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				pair.GetQuoteTokenDenom(), 1_000_000))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the position")
			liquidator := sample.AccAddress()
			err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)

			if tc.expectedPass {
				require.NoError(t, err)

				// We effectively closed the position
				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.Equal(t, sdk.ZeroDec(), newPosition.Size_)
				require.Equal(t, sdk.ZeroDec(), newPosition.Margin)
				require.Equal(t, sdk.ZeroDec(), newPosition.OpenNotional)

				// liquidator fee is half the liquidation fee of ExchangedQuoteAssetAmount
				// feeToLiquidator = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
				liquidatorBalance := nibiruApp.BankKeeper.GetBalance(
					ctx, liquidator, pair.GetQuoteTokenDenom())
				assert.Equal(t, tc.expectedFeeToLiquidator.String(), liquidatorBalance.String())

				perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
					types.PerpEFModuleAccount)
				perpEFBalance := nibiruApp.BankKeeper.GetBalance(
					ctx, perpEFAddr, pair.GetQuoteTokenDenom())
				require.Equal(t, tc.expectedPerpEFBalance.String(), perpEFBalance.String())
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
