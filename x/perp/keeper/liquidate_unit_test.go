package keeper

import (
	"math"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"

	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestDistributeLiquidateRewards_Error(t *testing.T) {
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

				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             "",
					},
				)
				require.Error(t, err)
			},
		},
		{
			name: "invalid pair - error",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				liquidator := sample.AccAddress()
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
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
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, BtcNusdPair).Return(false)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: BtcNusdPair.String(),
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

func TestDistributeLiquidateRewards_Happy(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "healthy liquidation",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidator := sample.AccAddress()

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, BtcNusdPair).Return(true)

				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))

				mocks.mockBankKeeper.
					EXPECT().GetBalance(ctx, authtypes.NewModuleAddress(types.VaultModuleAccount), "NUSD").
					Return(sdk.NewCoin("NUSD", sdk.NewInt(math.MaxInt64)))
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewCoin("NUSD", sdk.OneInt())),
				).Return(nil)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidator,
					sdk.NewCoins(sdk.NewCoin("NUSD", sdk.OneInt())),
				).Return(nil)

				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{
						BadDebt:                sdk.OneDec(),
						FeeToLiquidator:        sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: BtcNusdPair.String(),
							}},
					},
				)
				require.NoError(t, err)
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

func TestExecuteFullLiquidation_UnitWithMocks(t *testing.T) {
	tests := []struct {
		name string

		liquidationFee      sdk.Dec
		initialPositionSize sdk.Dec
		initialMargin       sdk.Dec
		initialOpenNotional sdk.Dec

		// amount of quote obtained by trading <initialPositionSize> base
		baseAssetPriceInQuote sdk.Dec

		expectedLiquidationBadDebt     sdk.Dec
		expectedFundsToPerpEF          sdk.Int
		expectedFundsToLiquidator      sdk.Int
		expectedExchangedNotionalValue sdk.Dec
		expectedMarginToVault          sdk.Dec
		expectedPositionRealizedPnl    sdk.Dec
		expectedPositionBadDebt        sdk.Dec
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

			liquidationFee:      sdk.MustNewDecFromStr("0.1"),
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:     sdk.ZeroDec(),
			expectedFundsToPerpEF:          sdk.NewInt(5),
			expectedFundsToLiquidator:      sdk.NewInt(5),
			expectedExchangedNotionalValue: sdk.NewDec(100),
			expectedMarginToVault:          sdk.NewDec(-10),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionBadDebt:        sdk.ZeroDec(),
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

			liquidationFee:      sdk.MustNewDecFromStr("0.3"),
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:     sdk.NewDec(5),
			expectedFundsToPerpEF:          sdk.ZeroInt(),
			expectedFundsToLiquidator:      sdk.NewInt(15),
			expectedExchangedNotionalValue: sdk.NewDec(100),
			expectedMarginToVault:          sdk.NewDec(-10),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionBadDebt:        sdk.ZeroDec(),
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
				- liquidator gets 80 NUSD * 0.3 / 2 = 12 NUSD
				- position has zero margin, so all of liquidation fee is bad debt
				- ecosystem fund gets nothing (0 NUSD)
			*/
			name: "position has + margin and bad debt - 1",

			liquidationFee:      sdk.MustNewDecFromStr("0.3"),
			initialPositionSize: sdk.NewDec(100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(80), // price dropped

			expectedLiquidationBadDebt:     sdk.NewDec(22),
			expectedFundsToPerpEF:          sdk.ZeroInt(),
			expectedFundsToLiquidator:      sdk.NewInt(12),
			expectedExchangedNotionalValue: sdk.NewDec(80),
			expectedMarginToVault:          sdk.ZeroDec(),
			expectedPositionRealizedPnl:    sdk.NewDec(-20),
			expectedPositionBadDebt:        sdk.NewDec(10),
		},
		{
			/*
				- short position
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

			liquidationFee:      sdk.MustNewDecFromStr("0.1"),
			initialPositionSize: sdk.NewDec(-100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:     sdk.ZeroDec(),
			expectedFundsToPerpEF:          sdk.NewInt(5),
			expectedFundsToLiquidator:      sdk.NewInt(5),
			expectedExchangedNotionalValue: sdk.NewDec(100),
			expectedMarginToVault:          sdk.NewDec(-10),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionBadDebt:        sdk.ZeroDec(),
		},
		{
			/*
				- short position
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

			liquidationFee:      sdk.MustNewDecFromStr("0.3"),
			initialPositionSize: sdk.NewDec(-100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedLiquidationBadDebt:     sdk.NewDec(5),
			expectedFundsToPerpEF:          sdk.ZeroInt(),
			expectedFundsToLiquidator:      sdk.NewInt(15),
			expectedExchangedNotionalValue: sdk.NewDec(100),
			expectedMarginToVault:          sdk.NewDec(-10),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionBadDebt:        sdk.ZeroDec(),
		},
		{
			/*
				- short position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)
				- BTC increases in price (1 BTC = 1.2 NUSD)
				- position notional of 120 NUSD
				- unrealized PnL of -20 NUSD, wipes out margin

				- position has zero margin remaining
				- position has bad debt
				- no funding payment

				- liquidation fee ratio is 0.3
				- notional exchanged is 120 NUSD
				- liquidator gets 120 NUSD * 0.3 / 2 = 18 NUSD
				- position has zero margin, so all of liquidation fee is bad debt
				- ecosystem fund gets nothing (0 NUSD)
			*/
			name: "position has + margin and bad debt - 2",

			liquidationFee:      sdk.MustNewDecFromStr("0.3"),
			initialPositionSize: sdk.NewDec(-100),
			initialMargin:       sdk.NewDec(10),
			initialOpenNotional: sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(120), // price increased

			expectedLiquidationBadDebt:     sdk.NewDec(28),
			expectedFundsToPerpEF:          sdk.ZeroInt(),
			expectedFundsToLiquidator:      sdk.NewInt(18),
			expectedExchangedNotionalValue: sdk.NewDec(120),
			expectedMarginToVault:          sdk.ZeroDec(),
			expectedPositionRealizedPnl:    sdk.NewDec(-20),
			expectedPositionBadDebt:        sdk.NewDec(10),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("setup variables")
			perpKeeper, mocks, ctx := getKeeper(t)
			liquidatorAddr := sample.AccAddress()
			traderAddr := sample.AccAddress()
			baseAssetDirection := vpooltypes.Direction_ADD_TO_POOL
			if tc.initialPositionSize.IsNegative() {
				baseAssetDirection = vpooltypes.Direction_REMOVE_FROM_POOL
			}

			t.Log("mock bank keeper")
			if tc.expectedFundsToPerpEF.IsPositive() {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewCoin("NUSD", tc.expectedFundsToPerpEF)),
				).Return(nil)
			}
			if tc.expectedFundsToLiquidator.IsPositive() {
				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))
				mocks.mockBankKeeper.
					EXPECT().GetBalance(ctx, authtypes.NewModuleAddress(types.VaultModuleAccount), "NUSD").
					Return(sdk.NewCoin("NUSD", sdk.NewInt(math.MaxInt64)))
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidatorAddr,
					sdk.NewCoins(sdk.NewCoin("NUSD", tc.expectedFundsToLiquidator)),
				).Return(nil)
			}
			expectedTotalBadDebtInt := tc.expectedLiquidationBadDebt.RoundInt()
			if expectedTotalBadDebtInt.IsPositive() {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount,
					sdk.NewCoins(sdk.NewCoin("NUSD", expectedTotalBadDebtInt)),
				)
			}

			t.Log("setup perp keeper params")
			newParams := types.DefaultParams()
			newParams.LiquidationFeeRatio = tc.liquidationFee
			perpKeeper.SetParams(ctx, newParams)
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: BtcNusdPair.String(),
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(), // zero funding payment for this test case
				},
			})

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, BtcNusdPair).AnyTimes().Return(true)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					BtcNusdPair,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					BtcNusdPair,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
					/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
				).Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockVpoolKeeper.EXPECT().
				GetSpotPrice(ctx, BtcNusdPair).
				Return(sdk.OneDec(), nil)

			t.Log("create and set the initial position")
			position := types.Position{
				TraderAddress:                       traderAddr.String(),
				Pair:                                BtcNusdPair.String(),
				Size_:                               tc.initialPositionSize,
				Margin:                              tc.initialMargin,
				OpenNotional:                        tc.initialOpenNotional,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         ctx.BlockHeight(),
			}
			perpKeeper.SetPosition(ctx, BtcNusdPair, traderAddr, &position)

			t.Log("execute full liquidation")
			liquidationResp, err := perpKeeper.ExecuteFullLiquidation(
				ctx, liquidatorAddr, &position)
			require.NoError(t, err)

			t.Log("assert liquidation response fields")
			assert.EqualValues(t, tc.expectedLiquidationBadDebt, liquidationResp.BadDebt)
			assert.EqualValues(t, tc.expectedFundsToLiquidator, liquidationResp.FeeToLiquidator)
			assert.EqualValues(t, tc.expectedFundsToPerpEF, liquidationResp.FeeToPerpEcosystemFund)
			assert.EqualValues(t, liquidatorAddr.String(), liquidationResp.Liquidator)

			t.Log("assert position response fields")
			positionResp := liquidationResp.PositionResp
			assert.EqualValues(t,
				tc.expectedExchangedNotionalValue,
				positionResp.ExchangedNotionalValue) // amount of quote exchanged
			// Initial position size is sold back to to vpool
			assert.EqualValues(t, tc.initialPositionSize.Neg(), positionResp.ExchangedPositionSize)
			// ( oldMargin + unrealizedPnL - fundingPayment ) * -1
			assert.EqualValues(t, tc.expectedMarginToVault, positionResp.MarginToVault)
			assert.EqualValues(t, tc.expectedPositionBadDebt, positionResp.BadDebt)
			assert.EqualValues(t, tc.expectedPositionRealizedPnl, positionResp.RealizedPnl)
			assert.True(t, positionResp.FundingPayment.IsZero())
			// Unrealized PnL should always be zero after a full close
			assert.True(t, positionResp.UnrealizedPnlAfter.IsZero())

			t.Log("assert new position fields")
			newPosition := positionResp.Position
			assert.EqualValues(t, traderAddr.String(), newPosition.TraderAddress)
			assert.EqualValues(t, BtcNusdPair.String(), newPosition.Pair)
			assert.True(t, newPosition.Size_.IsZero())        // always zero
			assert.True(t, newPosition.Margin.IsZero())       // always zero
			assert.True(t, newPosition.OpenNotional.IsZero()) // always zero
			assert.True(t, newPosition.LastUpdateCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  BtcNusdPair.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
				ExchangedPositionSize: positionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       sdk.NewCoin(BtcNusdPair.GetQuoteTokenDenom(), tc.expectedFundsToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(BtcNusdPair.GetQuoteTokenDenom(), tc.expectedFundsToPerpEF),
				BadDebt:               tc.expectedLiquidationBadDebt,
				Margin:                sdk.NewCoin(BtcNusdPair.GetQuoteTokenDenom(), newPosition.Margin.RoundInt()),
				PositionNotional:      positionResp.PositionNotional,
				PositionSize:          newPosition.Size_,
				UnrealizedPnl:         positionResp.UnrealizedPnlAfter,
				MarkPrice:             sdk.OneDec(),
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}
