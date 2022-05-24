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
			traderFunds:    sdk.NewInt64Coin("NUSD", 60),
		},
		{
			name:           "liquidateEmptyPositionSELL",
			side:           types.Side_SELL,
			quote:          sdk.NewInt(0),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 60),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)
			pair := common.TokenPair("BTC:NUSD")

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
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice)
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
			_, err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)

			require.Error(t, err)

			// No change in the position
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice)
			require.Equal(t, position.Size_, newPosition.Size_)
			require.Equal(t, position.Margin, newPosition.Margin)
			require.Equal(t, position.OpenNotional, newPosition.OpenNotional)
		})
	}
}

func TestExecuteFullLiquidation(t *testing.T) {
	// constants for this suite
	pair := common.TokenPair("BTC:NUSD")
	alice := sample.AccAddress()

	testCases := []struct {
		name                      string
		positionSide              types.Side
		quoteAmount               sdk.Int
		leverage                  sdk.Dec
		baseAssetLimit            sdk.Dec
		liquidationFee            sdk.Dec
		traderFunds               sdk.Coin
		expectedLiquidatorBalance sdk.Coin
		expectedPerpEFBalance     sdk.Coin
		expectedBadDebt           sdk.Dec
		expectedEvent             sdk.Event
	}{
		{
			name:           "happy path - Buy",
			positionSide:   types.Side_BUY,
			quoteAmount:    sdk.NewInt(50_000),
			leverage:       sdk.OneDec(),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 50_100),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.1 / 2 = 2500
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 2_500),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_047_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: alice, Pair: pair.String(),
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
			positionSide:   types.Side_SELL,
			quoteAmount:    sdk.NewInt(50_000),
			traderFunds:    sdk.NewInt64Coin("NUSD", 50_100),
			leverage:       sdk.OneDec(),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.123123"),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.123123 / 2 = 3078.025 → 3078
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 3078),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_046_972),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: alice, Pair: pair.String(),
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
			positionSide:   types.Side_BUY,
			quoteAmount:    sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 25_000),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: alice,
						Pair:          pair.String(),
						Margin:        sdk.ZeroDec(),
						OpenNotional:  sdk.ZeroDec(),
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
			positionSide:   types.Side_SELL,
			quoteAmount:    sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 25_000),
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: alice, Pair: pair.String(),
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
			t.Log("Initialize variables")
			nibiruApp, ctx := testutil.NewNibiruApp(true)
			vpoolKeeper := &nibiruApp.VpoolKeeper
			perpKeeper := &nibiruApp.PerpKeeper
			liquidator := sample.AccAddress()
			var err error

			t.Log("Create vpool")
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

			t.Log("Set perp params")
			params := types.DefaultParams()
			params.LiquidationFee = tc.liquidationFee.MulInt64(1_000_000).RoundInt64()
			perpKeeper.SetParams(ctx, params)
			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       pair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})

			t.Log("Fund trader (Alice) account with sufficient quote")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
				sdk.NewCoins(tc.traderFunds)))

			t.Log("Open position")
			require.NoError(t, nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.positionSide, alice, tc.quoteAmount, tc.leverage, tc.baseAssetLimit))

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice)
			require.NoError(t, err)

			t.Log("Fund vault and PerpEF")
			startingModuleFunds := sdk.NewCoins(
				sdk.NewInt64Coin(pair.GetQuoteTokenDenom(), 1_000_000),
			)
			require.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			require.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (entire) position")
			_, err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)
			require.NoError(t, err)

			t.Log("Check events")
			assert.Contains(t, ctx.EventManager().Events(), tc.expectedEvent)

			t.Log("Check new position")
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice)
			assert.True(t, newPosition.Size_.IsZero())
			assert.True(t, newPosition.Margin.IsZero())
			assert.True(t, newPosition.OpenNotional.IsZero())

			t.Log("Check liquidator balance")
			assert.EqualValues(t,
				tc.expectedLiquidatorBalance,
				nibiruApp.BankKeeper.GetBalance(
					ctx,
					liquidator,
					pair.GetQuoteTokenDenom(),
				),
			)

			t.Log("Check PerpEF balance")
			require.EqualValues(t,
				tc.expectedPerpEFBalance.String(),
				nibiruApp.BankKeeper.GetBalance(
					ctx,
					nibiruApp.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount),
					pair.GetQuoteTokenDenom(),
				).String(),
			)
		})
	}
}
