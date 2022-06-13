package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"

	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
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
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)
			pair, err2 := common.NewAssetPairFromStr("BTC:NUSD")
			require.NoError(t, err2)

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				pair,
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

			t.Log("Fund trader account with sufficient quote")
			var err error
			trader := sample.AccAddress()
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
				sdk.NewCoins(tc.traderFunds))
			require.NoError(t, err)

			t.Log("Open position")
			err = nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.side, trader, tc.quote, tc.leverage, tc.baseLimit)

			require.NoError(t, err)

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
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
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
			assert.Equal(t, position.Size_, newPosition.Size_)
			assert.Equal(t, position.Margin, newPosition.Margin)
			assert.Equal(t, position.OpenNotional, newPosition.OpenNotional)
		})
	}
}

func TestExecuteFullLiquidation(t *testing.T) {
	// constants for this suite
	pair, err := common.NewAssetPairFromStr("BTC:NUSD")
	require.NoError(t, err)

	trader := sample.AccAddress()

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
			// There's a 20 bps tx fee on open position.
			// This tx fee is split 50/50 bw the PerpEF and Treasury.
			// txFee = exchangedQuote * 20 bps = 100
			traderFunds: sdk.NewInt64Coin("NUSD", 50_100),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.1 / 2 = 2500
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 2_500),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_047_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: trader.String(), Pair: pair.String(),
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
			name:         "happy path - Sell",
			positionSide: types.Side_SELL,
			quoteAmount:  sdk.NewInt(50_000),
			// There's a 20 bps tx fee on open position.
			// This tx fee is split 50/50 bw the PerpEF and Treasury.
			// txFee = exchangedQuote * 20 bps = 100
			traderFunds:    sdk.NewInt64Coin("NUSD", 50_100),
			leverage:       sdk.OneDec(),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.123123"),
			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			//   = 50_000 * 0.123123 / 2 = 3078.025 → 3078
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 3078),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_046_972),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: trader.String(), Pair: pair.String(),
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
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: trader.String(),
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
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
			expectedEvent: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress: trader.String(), Pair: pair.String(),
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
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				pair,
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

			t.Log("Fund trader account with sufficient quote")
			var err error
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
				sdk.NewCoins(tc.traderFunds))
			require.NoError(t, err)

			t.Log("Open position")
			err = nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.positionSide, trader, tc.quoteAmount, tc.leverage, tc.baseAssetLimit)
			require.NoError(t, err)

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
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
			_, err = nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidator, position)
			require.NoError(t, err)

			t.Log("Verify expected values using internal event due to usage of private fns")
			assert.Contains(t, ctx.EventManager().Events(), tc.expectedEvent)

			t.Log("Check correctness of new position")
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
			assert.Equal(t, sdk.ZeroDec(), newPosition.Size_)
			assert.True(t, newPosition.Margin.Equal(sdk.NewDec(0)))
			assert.True(t, newPosition.OpenNotional.Equal(sdk.NewDec(0)))

			t.Log("Check correctness of liquidation fee distributions")
			liquidatorBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, liquidator, pair.GetQuoteTokenDenom())
			assert.Equal(t, tc.expectedLiquidatorBalance.String(), liquidatorBalance.String())

			perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
				types.PerpEFModuleAccount)
			perpEFBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, perpEFAddr, pair.GetQuoteTokenDenom())
			require.Equal(t, tc.expectedPerpEFBalance.String(), perpEFBalance.String())
		})
	}
}

func TestExecutePartialLiquidation_EmptyPosition(t *testing.T) {
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
			t.Log("Initialize keepers, pair, and liquidator")
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)
			pair, err := common.NewAssetPairFromStr("xxx:yyy")
			require.NoError(t, err)
			vpoolKeeper := &nibiruApp.VpoolKeeper
			perpKeeper := &nibiruApp.PerpKeeper

			t.Log("Create vpool")
			vpoolKeeper.CreatePool(
				ctx,
				pair,
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

			t.Log("Fund trader account with sufficient quote")
			trader := sample.AccAddress()
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
				sdk.NewCoins(tc.traderFunds)))

			t.Log("Open position")
			require.NoError(t, nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.side, trader, tc.quote, tc.leverage, tc.baseLimit))

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
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
			_, err = nibiruApp.PerpKeeper.ExecutePartialLiquidation(ctx, liquidator, position)

			require.Error(t, err)

			// No change in the position
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
			require.Equal(t, position.Size_, newPosition.Size_)
			require.Equal(t, position.Margin, newPosition.Margin)
			require.Equal(t, position.OpenNotional, newPosition.OpenNotional)
		})
	}
}

func TestExecutePartialLiquidation(t *testing.T) {
	// constants for this suite
	pair, err := common.NewAssetPairFromStr("xxx:yyy")
	require.NoError(t, err)

	trader := sample.AccAddress()
	partialLiquidationRatio := sdk.MustNewDecFromStr("0.4")

	testCases := []struct {
		name                      string
		side                      types.Side
		quote                     sdk.Int
		leverage                  sdk.Dec
		baseLimit                 sdk.Dec
		liquidationFee            sdk.Dec
		traderFunds               sdk.Coin
		expectedLiquidatorBalance sdk.Coin
		expectedPerpEFBalance     sdk.Coin
		expectedBadDebt           sdk.Dec

		expectedPositionSize    sdk.Dec
		expectedMarginRemaining sdk.Dec

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
			/* expectedPositionSize =  */
			// 24_999.9999999875000000001 * 0.6
			expectedPositionSize:    sdk.MustNewDecFromStr("14999.999999925000000001"),
			expectedMarginRemaining: sdk.MustNewDecFromStr("47999.999999994000000000"), // approx 2k less but slippage

			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * 0.4 * liquidationFee / 2
			//   = 50_000 * 0.4 * 0.1 / 2 = 1_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("yyy", 1_000),

			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress:                       trader.String(),
						Pair:                                pair.String(),
						Size_:                               sdk.MustNewDecFromStr("14999.999999925000000001"),
						Margin:                              sdk.MustNewDecFromStr("50000"),
						OpenNotional:                        sdk.MustNewDecFromStr("29999.999999940000000001"),
						LastUpdateCumulativePremiumFraction: sdk.OneDec(),
						BlockNumber:                         1,
					},
					ExchangedQuoteAssetAmount: sdk.MustNewDecFromStr("20000.000000059999999999"),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("-9999.999999950000000000"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.ZeroDec(),
					MarginToVault:             sdk.ZeroDec(),
					UnrealizedPnlAfter:        sdk.MustNewDecFromStr("0.000000000000000001"),
				},
				/* function */ "decrease_position",
			),
		},
		{
			name:           "happy path - Sell",
			side:           types.Side_SELL,
			quote:          sdk.NewInt(50_000),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 50_100),
			// There's a 20 bps tx fee on open position.
			// This tx fee is split 50/50 bw the PerpEF and Treasury.
			// exchangedQuote * 20 bps = 100

			expectedPositionSize:    sdk.MustNewDecFromStr("-15000.000000115000000001"), // ~-25k * 0.6
			expectedMarginRemaining: sdk.MustNewDecFromStr("48000.000000014000000000"),  // approx 2k less but slippage

			// feeToLiquidator
			//   = positionResp.ExchangedQuoteAssetAmount * 0.4 * liquidationFee / 2
			//   = 50_000 * 0.4 * 0.1 / 2 = 1_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("yyy", 1_000),

			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						TraderAddress:                       trader.String(),
						Pair:                                pair.String(),
						Size_:                               sdk.MustNewDecFromStr("-15000.000000115000000001"),
						Margin:                              sdk.MustNewDecFromStr("50000"),
						OpenNotional:                        sdk.MustNewDecFromStr("30000.000000140000000000"),
						LastUpdateCumulativePremiumFraction: sdk.OneDec(),
						BlockNumber:                         1,
					},
					ExchangedQuoteAssetAmount: sdk.MustNewDecFromStr("19999.999999860000000000"),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("10000.000000010000000000"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.ZeroDec(),
					MarginToVault:             sdk.ZeroDec(),
					UnrealizedPnlAfter:        sdk.MustNewDecFromStr("-0.000000000000000001"),
				},
				/* function */ "decrease_position",
			),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutilapp.NewNibiruApp(true)

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				pair,
				/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
				/* quoteAssetReserves */ sdk.NewDec(10_000_000_000_000_000),
				/* baseAssetReserves */ sdk.NewDec(5_000_000_000_000_000),
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
				partialLiquidationRatio,
			))

			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       pair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Fund trader account with sufficient quote")
			var err error
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, trader,
				sdk.NewCoins(tc.traderFunds))
			require.NoError(t, err)

			t.Log("increment block height and time for TWAP calculation")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
				WithBlockTime(time.Now().Add(time.Minute))

			t.Log("Open position")
			err = nibiruApp.PerpKeeper.OpenPosition(
				ctx, pair, tc.side, trader, tc.quote, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("Get the position")
			position, err := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
			require.NoError(t, err)

			t.Log("Artificially populate Vault and PerpEF to prevent BankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				pair.GetQuoteTokenDenom(), 1_000_000))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (partial) position")
			liquidator := sample.AccAddress()
			_, err = nibiruApp.PerpKeeper.ExecutePartialLiquidation(ctx, liquidator, position)
			require.NoError(t, err)

			t.Log("Verify expected values using internal event due to usage of private fns")
			assert.Contains(t, ctx.EventManager().Events(), tc.internal_position_response_event)

			t.Log("Check correctness of new position")
			newPosition, _ := nibiruApp.PerpKeeper.GetPosition(ctx, pair, trader)
			assert.Equal(t, tc.expectedPositionSize, newPosition.Size_)
			assert.Equal(t, tc.expectedMarginRemaining, newPosition.Margin)

			t.Log("Check liquidator balance")
			assert.EqualValues(t,
				tc.expectedLiquidatorBalance.String(),
				nibiruApp.BankKeeper.GetBalance(
					ctx,
					liquidator,
					pair.GetQuoteTokenDenom(),
				).String(),
			)

			t.Log("Check PerpEF balance")
			perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
				types.PerpEFModuleAccount)
			assert.EqualValues(t, perpEFAddr, nibiruApp.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount))
			assert.EqualValues(t,
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
