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

func TestExecuteFullLiquidation_EmptyPosition(t *testing.T) {
	testCases := []struct {
		name           string
		side           types.Side
		quote          sdk.Int
		leverage       sdk.Dec
		baseLimit      sdk.Dec
		liquidationFee sdk.Dec
		traderFunds    sdk.Coin
	}{
		{
			name:           "liquidateEmptyPositionBUY",
			side:           types.Side_BUY,
			quote:          sdk.NewInt(0),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 60),
		},
		{
			name:           "liquidateEmptyPositionSELL",
			side:           types.Side_SELL,
			quote:          sdk.NewInt(0),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 60),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)
			pair := common.TokenPair("xxx:yyy")

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
			alice := sample.AccAddress()
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

			t.Log("Liquidate the position")
			liquidator := sample.AccAddress()
			err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)

			require.Error(t, err)

			// No change in the position
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
			assert.Equal(t, position.Size_, newPosition.Size_)
			assert.Equal(t, position.Margin, newPosition.Margin)
			assert.Equal(t, position.OpenNotional, newPosition.OpenNotional)
		})
	}
}

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
			debt (2500 due to liquidator minus 50).
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
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 975_550),
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
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 975_550),
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
			assert.Equal(t, sdk.ZeroDec(), newPosition.Size_)
			assert.True(t, newPosition.Margin.Equal(sdk.NewDec(0)))
			assert.True(t, newPosition.OpenNotional.Equal(sdk.NewDec(0)))

			t.Log("Check correctness of liquidation fee distributions")
			liquidatorBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, liquidator, pair.GetQuoteTokenDenom())
			assert.Equal(t, tc.expectedFeeToLiquidator.String(), liquidatorBalance.String())

			perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
				types.PerpEFModuleAccount)
			perpEFBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, perpEFAddr, pair.GetQuoteTokenDenom())
			assert.Equal(t, tc.expectedPerpEFBalance.String(), perpEFBalance.String())
		})
	}
}

func TestCreatePartialLiquidation(t *testing.T) {
	testcases := []struct {
		name                    string
		side                    types.Side
		quote                   sdk.Int
		leverage                sdk.Dec
		baseLimit               sdk.Dec
		liquidationFee          sdk.Dec
		partialLiquidationRatio sdk.Dec
		removeMargin            sdk.Dec
		traderFunds             sdk.Int
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
			quote:                   sdk.NewInt(5000),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.5"),
			traderFunds:             sdk.NewInt(6000),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("1250"),
			// newPositionMargin = quote - expectedFee = 5000 - 250
			newPositionMargin:       sdk.NewDec(4750),
			newPositionOpenNotional: sdk.MustNewDecFromStr("2500"),
			expectedFee:             sdk.NewDec(250),
		},
		{
			name:                    "happy path - Sell",
			side:                    types.Side_SELL,
			quote:                   sdk.NewInt(5000),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.5"),
			traderFunds:             sdk.NewInt(6000),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("-1250"),
			// newPositionMargin = quote - expectedFee = 5000 - 250
			newPositionMargin:       sdk.NewDec(4750),
			newPositionOpenNotional: sdk.MustNewDecFromStr("2500"),
			expectedFee:             sdk.NewDec(250),
		},
		{
			name:                    "happy path - SellDifferentPercentage",
			side:                    types.Side_SELL,
			quote:                   sdk.NewInt(5000),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.4"),
			traderFunds:             sdk.NewInt(6000),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			expectedPass:            true,

			newPositionSize: sdk.MustNewDecFromStr("-1500"),
			// newPositionMargin = quote - expectedFee = 5000 - 200
			newPositionMargin:       sdk.NewDec(4800),
			newPositionOpenNotional: sdk.MustNewDecFromStr("3000"),
			expectedFee:             sdk.NewDec(200),
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
				sdk.NewCoins(sdk.NewCoin("yyy", tc.traderFunds)))
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
				ctx, position)

			if tc.expectedPass {
				require.NoError(t, err)

				t.Log("Verify successful setup of new position")
				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				assert.InDelta(t,
					/* expected */ tc.newPositionSize.MustFloat64(),
					/* actual */ newPosition.Size_.MustFloat64(),
					/* delta */ 0.0001)
				assert.EqualValues(t,
					/* expected */ tc.newPositionMargin.RoundInt().String(),
					/* actual */ newPosition.Margin.RoundInt().String(),
					/* delta */ 0.0001)
				assert.InDelta(t,
					/* expected */ tc.newPositionOpenNotional.MustFloat64(),
					/* actual */ newPosition.OpenNotional.MustFloat64(),
					/* delta */ 0.0001)

				t.Log("Verify correct values for liquidator fees and ExchangedQuoteAssetAmount")
				assert.EqualValues(t,
					tc.liquidationFee.
						Mul(liquidationOutput.PositionResp.ExchangedQuoteAssetAmount).
						QuoInt64(2).RoundInt().String(),
					liquidationOutput.FeeToLiquidator.RoundInt().String())
				assert.EqualValues(t,
					/* expected */ tc.expectedFee.QuoInt64(2).RoundInt().String(),
					/* actual */ liquidationOutput.FeeToLiquidator.RoundInt().String(),
				)
				assert.EqualValues(t,
					/* expected */ tc.expectedFee.QuoInt64(2).RoundInt().String(),
					/* actual */ liquidationOutput.FeeToPerpEcosystemFund.RoundInt().String(),
				)
			} else {
				t.Log("Verify error raised and position didn't change")
				require.Error(t, err)

				newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				assert.EqualValues(t, position.Size_, newPosition.Size_)
				assert.EqualValues(t, position.Margin, newPosition.Margin)
				assert.EqualValues(t, position.OpenNotional, newPosition.OpenNotional)
			}
		})
	}
}
