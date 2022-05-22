package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestExecuteFullLiquidation(t *testing.T) {
	// constants for this suite
	pair := common.TokenPair("xxx:yyy")
	alice := sample.AccAddress()

	testCases := []struct {
		name                             string
		side                             types.Side
		quote                            sdk.Int
		leverage                         sdk.Dec
		baseLimit                        sdk.Dec
		liquidationFee                   sdk.Dec
		traderFunds                      sdk.Coin
		expectedFeeToLiquidator          sdk.Coin
		expectedPerpEFBalance            sdk.Coin
		excpectedBadDebt                 sdk.Dec
		internal_position_response_event sdk.Event
	}{
		{
			name:           "happy path - Buy",
			side:           types.Side_BUY,
			quote:          sdk.NewInt(50_000),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 50_100),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.1 / 2 = 2500
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 2_500),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_045_050),
			excpectedBadDebt:      sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50_000),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("-24875.621890547263681592"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.ZeroDec(),
					MarginToVault:             sdk.NewDec(-50_000),
					UnrealizedPnlAfter:        sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
		{
			name:           "happy path - Sell",
			side:           types.Side_SELL,
			quote:          sdk.NewInt(50_000),
			traderFunds:    sdk.NewInt64Coin("yyy", 50_100),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.123123"),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.123123 / 2 = 3078.025 → 3078
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 3078),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_043_894),
			excpectedBadDebt:      sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50_000),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("25125.628140703517587940"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.MustNewDecFromStr("-0.000000000000000001"),
					MarginToVault:             sdk.MustNewDecFromStr("-49999.999999999999999999"),
					UnrealizedPnlAfter:        sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
		{
			/* We open a position for 500k, with a liquidation fee of 50k.
			This means 25k for the liquidator, and 25k for the perp fund.
			Because the user only have margin for 50, we create 24950 of bad
			debt (25000 due to liquidator minus 50).
			*/
			name:           "happy path - BadDebt, long",
			side:           types.Side_BUY,
			quote:          sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 950_600),
			excpectedBadDebt:      sdk.MustNewDecFromStr("24950"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(500_000),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("-238095.238095238095238095"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.ZeroDec(),
					MarginToVault:             sdk.NewDec(-50),
					UnrealizedPnlAfter:        sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
		{
			// Same as above case but for shorts
			name:           "happy path - BadDebt, short",
			side:           types.Side_SELL,
			quote:          sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 950_600),
			excpectedBadDebt:      sdk.MustNewDecFromStr("24950"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(500_000),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("263157.894736842105263158"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.ZeroDec(),
					MarginToVault:             sdk.NewDec(-50),
					UnrealizedPnlAfter:        sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				pair.String(),
				/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
				/* quoteAssetReserves */ sdk.NewDec(10_000_000),
				/* baseAssetReserves */ sdk.NewDec(5_000_000),
				/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("1"),
				/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("0.1"),
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
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
				sdk.NewCoins(tc.traderFunds))
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

			t.Log("Liquidate the (entire) position")
			liquidator := sample.AccAddress()
			err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)
			require.NoError(t, err)

			t.Log("Verify expected values using internal event due to usage of private fns")
			assert.Contains(t, ctx.EventManager().Events(), tc.internal_position_response_event)

			t.Log("Check correctness of new position")
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
			require.Equal(t, sdk.ZeroDec(), newPosition.Size_)
			require.True(t, newPosition.Margin.Equal(sdk.NewDec(0)))
			require.True(t, newPosition.OpenNotional.Equal(sdk.NewDec(0)))

			t.Log("Check correctness of liquidation fee distributions")
			liquidatorBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, liquidator, pair.GetQuoteTokenDenom())
			assert.Equal(t, tc.expectedFeeToLiquidator.String(), liquidatorBalance.String())

			perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
				types.PerpEFModuleAccount)
			perpEFBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, perpEFAddr, pair.GetQuoteTokenDenom())
			require.Equal(t, tc.expectedPerpEFBalance.String(), perpEFBalance.String())
		})
	}
}
