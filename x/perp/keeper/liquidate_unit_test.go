package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func Test_distributeLiquidateRewards_Error(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "empty LiquidateResponse fails validation - error",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{})
				require.Error(t, err)
				require.ErrorContains(t, err, "must not have nil fields")
			},
		},
		{
			name: "invalid liquidator - panic",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)

				require.Panics(t, func() {
					err := perpKeeper.distributeLiquidateRewards(ctx,
						types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
							FeeToPerpEcosystemFund: sdk.OneDec(),
							Liquidator:             sdk.AccAddress{},
						},
					)
					require.Error(t, err)
				})
			},
		},
		{
			name: "invalid pair - error",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				liquidator := sample.AccAddress()
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
						FeeToPerpEcosystemFund: sdk.OneDec(),
						Liquidator:             liquidator,
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: "dai:usdc:usdt",
							}},
					},
				)
				require.Error(t, err)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "vpool does not exist - error",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidator := sample.AccAddress()
				pair := common.TokenPair("BTC:NUSD")
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(false)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
						FeeToPerpEcosystemFund: sdk.OneDec(),
						Liquidator:             liquidator,
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: pair.String(),
							}},
					},
				)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func Test_distributeLiquidateRewards_Happy(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "healthy liquidation",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidator := sample.AccAddress()
				pair := common.TokenPair("BTC:NUSD")

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

				vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				perpEFAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.VaultModuleAccount).
					Return(vaultAddr)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.PerpEFModuleAccount).
					Return(perpEFAddr)

				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewCoin("NUSD", sdk.OneInt())),
				).Return(nil)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.PerpEFModuleAccount, liquidator,
					sdk.NewCoins(sdk.NewCoin("NUSD", sdk.OneInt())),
				).Return(nil)

				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
						FeeToPerpEcosystemFund: sdk.OneDec(),
						Liquidator:             liquidator,
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: pair.String(),
							}},
					},
				)
				require.NoError(t, err)

				expectedEvents := []sdk.Event{
					events.NewTransferEvent(
						/* coin */ sdk.NewCoin("NUSD", sdk.OneInt()),
						/* from */ vaultAddr.String(),
						/* to */ perpEFAddr.String(),
					),
					events.NewTransferEvent(
						/* coin */ sdk.NewCoin("NUSD", sdk.OneInt()),
						/* from */ perpEFAddr.String(),
						/* to */ liquidator.String(),
					),
				}
				for _, event := range expectedEvents {
					assert.Contains(t, ctx.EventManager().Events(), event)
				}
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestExecuteFullLiquidationWithMocks(t *testing.T) {
	tests := []struct {
		name string

		liquidationFee      int64
		initialPositionSize sdk.Dec
		initialMargin       sdk.Dec
		initialOpenNotional sdk.Dec

		baseAssetPriceInQuote sdk.Dec // amount of quote obtained by trading <initialPositionSize> base

		expectedLiquidationBadDebt        sdk.Dec
		expectedFundsToPerpEF             sdk.Dec
		expectedFundsToLiquidator         sdk.Dec
		expectedExchangedQuoteAssetAmount sdk.Dec
		expectedMarginToVault             sdk.Dec
		expectedPositionRealizedPnl       sdk.Dec
		expectedPositionBadDebt           sdk.Dec
	}{
		{
			/*
				- long position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)

				- remaining margin more than liquidation fee
				- position has zero bad debt
				- no funding payment

				- liquidation fee ratio is 0.1
				- notional exchanged is 100 NUSD
				- liquidator gets 100 NUSD * 0.1 / 2 = 5 NUSD
				- ecosystem fund gets remaining = 5 NUSD
			*/
			name: "remaining margin more than liquidation fee",

			liquidationFee:      100_000, // 0.1 liquidation fee
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:        sdk.ZeroDec(),
			expectedFundsToPerpEF:             sdk.NewDec(5),
			expectedFundsToLiquidator:         sdk.NewDec(5),
			expectedExchangedQuoteAssetAmount: sdk.NewDec(100),
			expectedMarginToVault:             sdk.NewDec(-10),
			expectedPositionRealizedPnl:       sdk.ZeroDec(),
			expectedPositionBadDebt:           sdk.ZeroDec(),
		},
		{
			/*
				- long position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)

				- remaining margin less than liquidation fee but greater than zero
				- position has zero bad debt
				- no funding payment

				- liquidation fee ratio is 0.3
				- notional exchanged is 100 NUSD
				- liquidator gets 100 NUSD * 0.3 / 2 = 15 NUSD
				- position only has 10 NUSD margin, so bad debt accrues
				- ecosystem fund gets nothing (0 NUSD)
			*/
			name: "remaining margin less than liquidation fee but greater than zero",

			liquidationFee:      300_000, // 0.3 liquidation fee
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:        sdk.NewDec(5),
			expectedFundsToPerpEF:             sdk.ZeroDec(),
			expectedFundsToLiquidator:         sdk.NewDec(15),
			expectedExchangedQuoteAssetAmount: sdk.NewDec(100),
			expectedMarginToVault:             sdk.NewDec(-10),
			expectedPositionRealizedPnl:       sdk.ZeroDec(),
			expectedPositionBadDebt:           sdk.ZeroDec(),
		},
		{
			/*
				- long position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)
				- BTC drops in price (1 BTC = 0.8 NUSD)
				- position notional of 80 NUSD
				- unrealized PnL of -20 NUSD, wipes out margin

				- position has zero margin remaining
				- position has bad debt
				- no funding payment

				- liquidation fee ratio is 0.3
				- notional exchanged is 80 NUSD
				- liquidator gets 80 NUSD * 0.3 / 2 = 25 NUSD
				- position has zero margin, so all of liquidation fee is bad debt
				- ecosystem fund gets nothing (0 NUSD)
			*/
			name: "position has zero margin and bad debt",

			liquidationFee:      300_000, // 0.3 liquidation fee
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(80), // price dropped

			expectedLiquidationBadDebt:        sdk.NewDec(22),
			expectedFundsToPerpEF:             sdk.ZeroDec(),
			expectedFundsToLiquidator:         sdk.NewDec(12),
			expectedExchangedQuoteAssetAmount: sdk.NewDec(80),
			expectedMarginToVault:             sdk.ZeroDec(),
			expectedPositionRealizedPnl:       sdk.NewDec(-20),
			expectedPositionBadDebt:           sdk.NewDec(10),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("setup variables")
			perpKeeper, mocks, ctx := getKeeper(t)
			liquidatorAddr := sample.AccAddress()
			traderAddr := sample.AccAddress()
			pair := common.TokenPair("BTC:NUSD")

			t.Log("mock account keeper")
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
			perpEFAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
			mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
				types.VaultModuleAccount).Return(vaultAddr)
			mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
				types.PerpEFModuleAccount).Return(perpEFAddr)

			t.Log("mock bank keeper")
			if tc.expectedFundsToPerpEF.IsPositive() {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewCoin("NUSD", tc.expectedFundsToPerpEF.RoundInt())),
				).Return(nil)
			}
			if tc.expectedFundsToLiquidator.IsPositive() {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.PerpEFModuleAccount, liquidatorAddr,
					sdk.NewCoins(sdk.NewCoin("NUSD", tc.expectedFundsToLiquidator.RoundInt())),
				).Return(nil)
			}

			t.Log("setup perp keeper params")
			newParams := types.DefaultParams()
			newParams.LiquidationFee = tc.liquidationFee
			perpKeeper.SetParams(ctx, newParams)
			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair: pair.String(),
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(), // zero funding payment for this test case
				},
			})

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					pair,
					vpooltypes.Direction_ADD_TO_POOL,
					/*baseAssetAmount=*/ tc.initialPositionSize,
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					pair,
					/*baseAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
					/*baseAssetAmount=*/ tc.initialPositionSize,
					/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
				).Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)

			t.Log("create and set the initial position")
			position := types.Position{
				Address:                             traderAddr.String(),
				Pair:                                pair.String(),
				Size_:                               tc.initialPositionSize,
				Margin:                              tc.initialMargin,
				OpenNotional:                        tc.initialOpenNotional,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				LiquidityHistoryIndex:               0,
				BlockNumber:                         ctx.BlockHeight(),
			}
			perpKeeper.SetPosition(ctx, pair, traderAddr.String(), &position)

			t.Log("execute full liquidation")
			liquidationResp, err := perpKeeper.ExecuteFullLiquidation(ctx, liquidatorAddr, &position)
			require.NoError(t, err)

			t.Log("assert liquidation response fields")
			assert.EqualValues(t, tc.expectedLiquidationBadDebt, liquidationResp.BadDebt)
			assert.EqualValues(t, tc.expectedFundsToLiquidator, liquidationResp.FeeToLiquidator)
			assert.EqualValues(t, tc.expectedFundsToPerpEF, liquidationResp.FeeToPerpEcosystemFund)
			assert.EqualValues(t, liquidatorAddr, liquidationResp.Liquidator)

			t.Log("assert position response fields")
			positionResp := liquidationResp.PositionResp
			assert.EqualValues(t, tc.expectedExchangedQuoteAssetAmount, positionResp.ExchangedQuoteAssetAmount) // amount of quote exchanged
			assert.EqualValues(t, tc.initialPositionSize.Neg(), positionResp.ExchangedPositionSize)             // sold back to vpool
			assert.EqualValues(t, tc.expectedMarginToVault, positionResp.MarginToVault)                         // ( oldMargin + unrealzedPnL - fundingPayment ) * -1
			assert.EqualValues(t, tc.expectedPositionBadDebt, positionResp.BadDebt)
			assert.EqualValues(t, tc.expectedPositionRealizedPnl, positionResp.RealizedPnl)
			assert.True(t, positionResp.FundingPayment.IsZero())
			assert.True(t, positionResp.UnrealizedPnlAfter.IsZero()) // always zero

			t.Log("assert new position fields")
			newPosition := positionResp.Position
			assert.EqualValues(t, traderAddr.String(), newPosition.Address)
			assert.EqualValues(t, pair.String(), newPosition.Pair)
			assert.True(t, newPosition.Size_.IsZero())        // always zero
			assert.True(t, newPosition.Margin.IsZero())       // always zero
			assert.True(t, newPosition.OpenNotional.IsZero()) // always zero
			assert.True(t, newPosition.LastUpdateCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, 0, newPosition.LiquidityHistoryIndex)
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)
		})
	}
}
