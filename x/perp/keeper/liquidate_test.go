package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	testutilevents "github.com/NibiruChain/nibiru/x/testutil"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"

	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"

	simapp2 "github.com/NibiruChain/nibiru/simapp"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestExecuteFullLiquidation(t *testing.T) {
	// constants for this suite
	tokenPair := common.MustNewAssetPair("BTC:NUSD")

	traderAddr := testutilevents.AccAddress()

	type test struct {
		positionSide              types.Side
		quoteAmount               sdk.Int
		leverage                  sdk.Dec
		baseAssetLimit            sdk.Dec
		liquidationFee            sdk.Dec
		traderFunds               sdk.Coin
		expectedLiquidatorBalance sdk.Coin
		expectedPerpEFBalance     sdk.Coin
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
			// startingBalance = 1* common.Precision
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_047_550),
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
			// startingBalance = 1* common.Precision
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("NUSD", 1_046_972),
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())
			perpKeeper := &nibiruApp.PerpKeeper

			t.Log("create vpool")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				/* quoteAssetReserves */ sdk.NewDec(10*common.Precision),
				/* baseAssetReserves */ sdk.NewDec(5*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

			nibiruApp.OracleKeeper.SetPrice(ctx, tokenPair, sdk.NewDec(2))

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
			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            tokenPair,
				LatestCumulativePremiumFraction: sdk.OneDec(),
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
				ctx, tokenPair, tc.positionSide, traderAddr, tc.quoteAmount, tc.leverage, tc.baseAssetLimit)
			require.NoError(t, err)

			t.Log("Artificially populate Vault and PerpEF to prevent bankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				tokenPair.QuoteDenom(), 1*common.Precision))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (entire) position")
			liquidatorAddr := testutilevents.AccAddress()
			liquidationResp, err := nibiruApp.PerpKeeper.ExecuteFullLiquidation(ctx, liquidatorAddr, positionResp.Position)
			require.NoError(t, err)

			t.Log("Check correctness of new position")
			newPosition, err := nibiruApp.PerpKeeper.Positions.Get(ctx, collections.Join(tokenPair, traderAddr))
			require.ErrorIs(t, err, collections.ErrNotFound)
			require.Empty(t, newPosition)

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
			newMarkPrice, err := vpoolKeeper.GetMarkPrice(ctx, tokenPair)
			require.NoError(t, err)
			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  tokenPair,
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

	traderAddr := testutilevents.AccAddress()
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

			// startingBalance = 1* common.Precision
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
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

			// startingBalance = 1* common.Precision
			// perpEFBalance = startingBalance + openPositionDelta + liquidateDelta
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 1_001_050),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := simapp2.NewTestNibiruAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tokenPair,
				/* quoteAssetReserves */ sdk.NewDec(10_000*common.Precision*common.Precision),
				/* baseAssetReserves */ sdk.NewDec(5_000*common.Precision*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
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

			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            tokenPair,
				LatestCumulativePremiumFraction: sdk.OneDec(),
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

			t.Log("Artificially populate Vault and PerpEF to prevent bankKeeper errors")
			startingModuleFunds := sdk.NewCoins(sdk.NewInt64Coin(
				tokenPair.QuoteDenom(), 1*common.Precision))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.VaultModuleAccount, startingModuleFunds))
			assert.NoError(t, simapp.FundModuleAccount(
				nibiruApp.BankKeeper, ctx, types.PerpEFModuleAccount, startingModuleFunds))

			t.Log("Liquidate the (partial) position")
			liquidator := testutilevents.AccAddress()
			liquidationResp, err := nibiruApp.PerpKeeper.ExecutePartialLiquidation(ctx, liquidator, positionResp.Position)
			require.NoError(t, err)

			t.Log("Check correctness of new position")
			newPosition, err := nibiruApp.PerpKeeper.Positions.Get(ctx, collections.Join(tokenPair, traderAddr))
			require.NoError(t, err)
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
			newMarkPrice, err := vpoolKeeper.GetMarkPrice(ctx, tokenPair)
			require.NoError(t, err)
			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  tokenPair,
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

func setPosition(k perpkeeper.Keeper, ctx sdk.Context, pos types.Position) {
	k.Positions.Insert(ctx, collections.Join(pos.Pair, sdk.MustAccAddressFromBech32(pos.TraderAddress)), pos)
}

func setPairMetadata(k perpkeeper.Keeper, ctx sdk.Context, pm types.PairMetadata) {
	k.PairsMetadata.Insert(ctx, pm.Pair, pm)
}
