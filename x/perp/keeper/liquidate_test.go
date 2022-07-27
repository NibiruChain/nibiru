package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestExecuteFullLiquidation(t *testing.T) {
	// constants for this suite
	tokenPair := common.MustNewAssetPair("BTC:NUSD")

	traderAddr := sample.AccAddress()

	type test struct {
		positionSide              types.Side
		quoteAmount               sdk.Int
		leverage                  sdk.Dec
		baseAssetLimit            sdk.Dec
		liquidationFee            sdk.Dec
		traderFunds               sdk.Coin
		expectedLiquidatorBalance sdk.Coin
		expectedPerpEFBalance     sdk.Coin
		expectedBadDebt           sdk.Dec
	}

	testCases := map[string]test{
		"happy path - Buy": {
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
			//   = positionResp.ExchangedNotionalValue * liquidationFee / 2
			//   = 50_000 * 0.1 / 2 = 2500
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 2_500),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_047_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
		},
		"happy path - Sell": {
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
			//   = positionResp.ExchangedNotionalValue * liquidationFee / 2
			//   = 50_000 * 0.123123 / 2 = 3078.025 → 3078
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 3078),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_046_972),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
		},
		"happy path - bad debt, long": {
			/* We open a position for 500k, with a liquidation fee of 50k.
			This means 25k for the liquidator, and 25k for the perp fund.
			Because the user only have margin for 50, we create 24950 of bad
			debt (25000 due to liquidator minus 50).
			*/
			positionSide:   types.Side_BUY,
			quoteAmount:    sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedNotionalValue * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 25_000),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
		},
		"happy path - bad debt, short": {
			// Same as above case but for shorts
			positionSide:   types.Side_SELL,
			quoteAmount:    sdk.NewInt(50),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseAssetLimit: sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("NUSD", 1150),
			// feeToLiquidator
			//   = positionResp.ExchangedNotionalValue * liquidationFee / 2
			//   = 500_000 * 0.1 / 2 = 25_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("NUSD", 25_000),
			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 975_550),
			expectedBadDebt:       sdk.MustNewDecFromStr("24950"),
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
			perpKeeper := &nibiruApp.PerpKeeper

			t.Log("create vpool")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
				/* quoteAssetReserves */ sdk.NewDec(10_000_000),
				/* baseAssetReserves */ sdk.NewDec(5_000_000),
				/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("1"),
				/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("0.1"),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))
			nibiruApp.PricefeedKeeper.ActivePairsStore().Set(ctx, tokenPair, true)

			t.Log("set perpkeeper params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, types.NewParams(
				params.Stopped,
				params.FeePoolFeeRatio,
				params.EcosystemFundFeeRatio,
				tc.liquidationFee,
				params.PartialLiquidationRatio,
				"hour",
				15*time.Minute,
			))
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       tokenPair,
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Fund trader account with sufficient quote")
			var err error
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
				sdk.NewCoins(tc.traderFunds))
			require.NoError(t, err)

			t.Log("Open position")
			positionResp, err := nibiruApp.PerpKeeper.OpenPosition(
				ctx, tokenPair, tc.positionSide, traderAddr, tc.quoteAmount, tc.leverage, tc.baseAssetLimit)
			require.NoError(t, err)

			t.Log("Artificially populate Vault and PerpEF to prevent BankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				tokenPair.QuoteDenom(), 1_000_000))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (entire) position")
			liquidatorAddr := sample.AccAddress()
			liquidationResp, err := nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidatorAddr, positionResp.Position)
			require.NoError(t, err)

			t.Log("Check correctness of new position")
			newPosition, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(tokenPair, traderAddr)
			require.ErrorIs(t, err, types.ErrPositionNotFound)
			require.Nil(t, newPosition)

			t.Log("Check correctness of liquidation fee distributions")
			liquidatorBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, liquidatorAddr, tokenPair.QuoteDenom())
			assert.EqualValues(t, tc.expectedLiquidatorBalance, liquidatorBalance)

			perpEFAddr := nibiruApp.AccountKeeper.GetModuleAddress(
				types.PerpEFModuleAccount)
			perpEFBalance := nibiruApp.BankKeeper.GetBalance(
				ctx, perpEFAddr, tokenPair.QuoteDenom())
			require.EqualValues(t, tc.expectedPerpEFBalance, perpEFBalance)

			t.Log("check emitted events")
			newMarkPrice, err := vpoolKeeper.GetSpotPrice(ctx, tokenPair)
			require.NoError(t, err)
			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  tokenPair.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  liquidationResp.PositionResp.ExchangedNotionalValue,
				ExchangedPositionSize: liquidationResp.PositionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.FeeToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.FeeToPerpEcosystemFund),
				BadDebt:               sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.BadDebt),
				Margin:                sdk.NewCoin(tokenPair.QuoteDenom(), sdk.ZeroInt()),
				PositionNotional:      liquidationResp.PositionResp.PositionNotional,
				PositionSize:          sdk.ZeroDec(),
				UnrealizedPnl:         liquidationResp.PositionResp.UnrealizedPnlAfter,
				MarkPrice:             newMarkPrice,
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestExecutePartialLiquidation(t *testing.T) {
	// constants for this suite
	tokenPair := common.MustNewAssetPair("xxx:yyy")

	traderAddr := sample.AccAddress()
	partialLiquidationRatio := sdk.MustNewDecFromStr("0.4")

	testCases := []struct {
		name           string
		side           types.Side
		quote          sdk.Int
		leverage       sdk.Dec
		baseLimit      sdk.Dec
		liquidationFee sdk.Dec
		traderFunds    sdk.Coin

		expectedLiquidatorBalance sdk.Coin
		expectedPerpEFBalance     sdk.Coin
		expectedBadDebt           sdk.Dec
		expectedPositionSize      sdk.Dec
		expectedMarginRemaining   sdk.Dec
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
			//   = positionResp.ExchangedNotionalValue * 0.4 * liquidationFee / 2
			//   = 50_000 * 0.4 * 0.1 / 2 = 1_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("yyy", 1_000),

			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
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
			//   = positionResp.ExchangedNotionalValue * 0.4 * liquidationFee / 2
			//   = 50_000 * 0.4 * 0.1 / 2 = 1_000
			expectedLiquidatorBalance: sdk.NewInt64Coin("yyy", 1_000),

			// startingBalance = 1_000_000
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
			expectedBadDebt:       sdk.MustNewDecFromStr("0"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"),
				/* quoteAssetReserves */ sdk.NewDec(10_000_000_000_000_000),
				/* baseAssetReserves */ sdk.NewDec(5_000_000_000_000_000),
				/* fluctuationLimitRatio */ sdk.MustNewDecFromStr("1"),
				/* maxOracleSpreadRatio */ sdk.MustNewDecFromStr("0.1"),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
			)
			nibiruApp.PricefeedKeeper.ActivePairsStore().Set(ctx, tokenPair, true)
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

			t.Log("Set vpool defined by pair on PerpKeeper")
			perpKeeper := &nibiruApp.PerpKeeper
			params := types.DefaultParams()

			perpKeeper.SetParams(ctx, types.NewParams(
				params.Stopped,
				params.FeePoolFeeRatio,
				params.EcosystemFundFeeRatio,
				tc.liquidationFee,
				partialLiquidationRatio,
				"hour",
				15*time.Minute,
			))

			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       tokenPair,
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Fund trader account with sufficient quote")
			var err error
			err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr,
				sdk.NewCoins(tc.traderFunds))
			require.NoError(t, err)

			t.Log("increment block height and time for TWAP calculation")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
				WithBlockTime(time.Now().Add(time.Minute))

			t.Log("Open position")
			positionResp, err := nibiruApp.PerpKeeper.OpenPosition(
				ctx, tokenPair, tc.side, traderAddr, tc.quote, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("Artificially populate Vault and PerpEF to prevent BankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				tokenPair.QuoteDenom(), 1_000_000))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (partial) position")
			liquidator := sample.AccAddress()
			liquidationResp, err := nibiruApp.PerpKeeper.ExecutePartialLiquidation(ctx, liquidator, positionResp.Position)
			require.NoError(t, err)

			t.Log("Check correctness of new position")
			newPosition, _ := nibiruApp.PerpKeeper.PositionsState(ctx).Get(tokenPair, traderAddr)
			assert.Equal(t, tc.expectedPositionSize, newPosition.Size_)
			assert.Equal(t, tc.expectedMarginRemaining, newPosition.Margin)

			t.Log("Check liquidator balance")
			assert.EqualValues(t,
				tc.expectedLiquidatorBalance.String(),
				nibiruApp.BankKeeper.GetBalance(
					ctx,
					liquidator,
					tokenPair.QuoteDenom(),
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
					tokenPair.QuoteDenom(),
				).String(),
			)

			t.Log("check emitted events")
			newMarkPrice, err := vpoolKeeper.GetSpotPrice(ctx, tokenPair)
			require.NoError(t, err)
			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  tokenPair.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  liquidationResp.PositionResp.ExchangedNotionalValue,
				ExchangedPositionSize: liquidationResp.PositionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidator.String(),
				FeeToLiquidator:       sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.FeeToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.FeeToPerpEcosystemFund),
				BadDebt:               sdk.NewCoin(tokenPair.QuoteDenom(), liquidationResp.BadDebt),
				Margin:                sdk.NewCoin(tokenPair.QuoteDenom(), newPosition.Margin.RoundInt()),
				PositionNotional:      liquidationResp.PositionResp.PositionNotional,
				PositionSize:          newPosition.Size_,
				UnrealizedPnl:         liquidationResp.PositionResp.UnrealizedPnlAfter,
				MarkPrice:             newMarkPrice,
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}
