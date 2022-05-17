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
		quote          sdk.Dec
		leverage       sdk.Dec
		baseLimit      sdk.Dec
		liquidationFee sdk.Dec
		traderFunds    sdk.Coin
	}{
		{
			name:           "liquidateEmptyPositionBUY",
			side:           types.Side_BUY,
			quote:          sdk.MustNewDecFromStr("0"),
			leverage:       sdk.OneDec(),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 60),
		},
		{
			name:           "liquidateEmptyPositionSELL",
			side:           types.Side_SELL,
			quote:          sdk.MustNewDecFromStr("0"),
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
			require.Equal(t, position.Size_, newPosition.Size_)
			require.Equal(t, position.Margin, newPosition.Margin)
			require.Equal(t, position.OpenNotional, newPosition.OpenNotional)
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
		quote                            sdk.Dec
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
			name:                    "happy path - Buy",
			side:                    types.Side_BUY,
			quote:                   sdk.NewDec(50_000),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			traderFunds:             sdk.NewInt64Coin("yyy", 50_100),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 2_500),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 1_045_050),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50),
					BadDebt:                   sdk.ZeroDec(),
					// ExchangedPositionSize:     sdk.MustNewDecFromStr("-24.999875000624996875"),
					ExchangedPositionSize: sdk.MustNewDecFromStr("-24.999875000624996875"),
					FundingPayment:        sdk.ZeroDec(),
					RealizedPnl:           sdk.ZeroDec(),
					MarginToVault:         sdk.NewDec(-50),
					UnrealizedPnlAfter:    sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
		{
			name:                    "happy path - Sell",
			side:                    types.Side_SELL,
			quote:                   sdk.NewDec(50_000),
			traderFunds:             sdk.NewInt64Coin("yyy", 50_100),
			leverage:                sdk.OneDec(),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.123123"),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 3078),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 1_043_893),
			excpectedBadDebt:        sdk.MustNewDecFromStr("0"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50),
					BadDebt:                   sdk.ZeroDec(),
					ExchangedPositionSize:     sdk.MustNewDecFromStr("25125628140703517587940"),
					FundingPayment:            sdk.ZeroDec(),
					RealizedPnl:               sdk.NewDec(-1),
					MarginToVault:             sdk.NewDec(-50),
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
			quote:          sdk.MustNewDecFromStr("50"),
			leverage:       sdk.MustNewDecFromStr("10000"),
			baseLimit:      sdk.ZeroDec(),
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
			traderFunds:    sdk.NewInt64Coin("yyy", 1150),
			// feeToLiquidator = positionResp.ExchangedQuoteAssetAmount * liquidationFee / 2
			// feeToLiquidator
			//   = liquidationAmount / 2 = quote * leverage / 2
			//   = 50 * 10_000 / 2 = 25_000
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			// perpEFBalance = startBalance - ... + ...
			//   = 1_000_000 - ... + ... = 975_500
			expectedPerpEFBalance: sdk.NewInt64Coin("yyy", 975_550),
			excpectedBadDebt:      sdk.MustNewDecFromStr("24950"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50),
					BadDebt:                   sdk.ZeroDec(),
					// ExchangedPositionSize:     sdk.MustNewDecFromStr("-23.8095238095238095238095"),
					ExchangedPositionSize: sdk.MustNewDecFromStr("-23.809523809523809523"),
					FundingPayment:        sdk.ZeroDec(),
					RealizedPnl:           sdk.ZeroDec(),
					MarginToVault:         sdk.NewDec(-50),
					UnrealizedPnlAfter:    sdk.ZeroDec(),
				},
				/* function */ "close_position_entirely",
			),
		},
		{
			// Same as above case but for shorts
			name:                    "happy path - BadDebt, short",
			side:                    types.Side_SELL,
			quote:                   sdk.MustNewDecFromStr("50"),
			leverage:                sdk.MustNewDecFromStr("10000"),
			baseLimit:               sdk.ZeroDec(),
			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			traderFunds:             sdk.NewInt64Coin("yyy", 1150),
			expectedFeeToLiquidator: sdk.NewInt64Coin("yyy", 25_000),
			expectedPerpEFBalance:   sdk.NewInt64Coin("yyy", 975_550),
			excpectedBadDebt:        sdk.MustNewDecFromStr("24950"),
			internal_position_response_event: events.NewInternalPositionResponseEvent(
				&types.PositionResp{
					Position: &types.Position{
						Address: alice.String(), Pair: pair.String(),
						Margin: sdk.ZeroDec(), OpenNotional: sdk.ZeroDec(),
					},
					ExchangedQuoteAssetAmount: sdk.NewDec(50),
					BadDebt:                   sdk.ZeroDec(),
					// ExchangedPositionSize:     sdk.MustNewDecFromStr("26.3157894736842105263158"),
					ExchangedPositionSize: sdk.MustNewDecFromStr("26.315789473684210526"),
					FundingPayment:        sdk.ZeroDec(),
					RealizedPnl:           sdk.ZeroDec(),
					MarginToVault:         sdk.NewDec(-50),
					UnrealizedPnlAfter:    sdk.ZeroDec(),
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
			// assert.Contains(t, ctx.EventManager().Events(), tc.internal_position_response_event)

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
		})
	}
}
