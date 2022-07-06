package keeper

import (
	"math"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestLiquidateIntoPartialLiquidation(t *testing.T) {
	tests := []struct {
		name string

		initialPositionSize         sdk.Dec
		initialPositionMargin       sdk.Dec
		initialPositionOpenNotional sdk.Dec

		newPositionNotional sdk.Dec
		exchangedSize       sdk.Dec
		exchangedNotional   sdk.Dec

		expectedLiquidatorFee sdk.Coin
		expectedPerpEFFee     sdk.Coin

		expectedPositionSize         sdk.Dec
		expectedPositionMargin       sdk.Dec
		expectedPositionOpenNotional sdk.Dec
		expectedUnrealizedPnl        sdk.Dec
	}{
		{
			name: "Partial Liquidation - just under maintenance margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(959),                // just below 6.25% margin ratio
			exchangedSize:       sdk.MustNewDecFromStr("0.5"),   // 1 * 0.5
			exchangedNotional:   sdk.MustNewDecFromStr("479.5"), // 959 * 0.5

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 3), // 959 * 0.5 * 0.0125 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 3), // 959 * 0.5 * 0.0125 / 2

			expectedPositionSize:         sdk.MustNewDecFromStr("0.5"),
			expectedPositionMargin:       sdk.MustNewDecFromStr("73.50625"), // 100 - 20.5 - 959*0.5*0.0125
			expectedPositionOpenNotional: sdk.NewDec(500),
			expectedUnrealizedPnl:        sdk.MustNewDecFromStr("-20.5"), // -41 * 0.5
		},
		{
			name: "Partial Liquidation - just above full liquidation",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.MustNewDecFromStr("911.3924051"),  // at 1.25% margin ratio
			exchangedSize:       sdk.MustNewDecFromStr("0.5"),          // 1 * 0.5
			exchangedNotional:   sdk.MustNewDecFromStr("455.69620255"), // 911.3924051 * 0.5

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 3), // 911.3924051 * 0.5 * 0.0125 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 3), // 911.3924051 * 0.5 * 0.0125 / 2

			expectedPositionSize:         sdk.MustNewDecFromStr("0.5"),
			expectedPositionMargin:       sdk.MustNewDecFromStr("50.000000018125"), // 100 - 88.60759494*0.5 - 911.3924051*0.5*0.0125
			expectedPositionOpenNotional: sdk.NewDec(500),
			expectedUnrealizedPnl:        sdk.MustNewDecFromStr("-44.30379745"), // -88.60759494 * 0.5
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := sample.AccAddress()
			liquidatorAddr := sample.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          common.PairBTCStable,
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			perpKeeper.SetPosition(ctx, common.PairBTCStable, traderAddr, &position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			})

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).Return(true).Times(2)
			mocks.mockVpoolKeeper.EXPECT().IsOverSpreadLimit(ctx, common.PairBTCStable).Return(false)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockVpoolKeeper.EXPECT().GetSpotPrice(ctx, common.PairBTCStable).Return(markPrice, nil)

			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.OneDec(),
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.OneDec(),
				).
				Return(tc.newPositionNotional, nil).Times(4)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.exchangedSize,
				).
				Return(tc.exchangedNotional, nil)
			mocks.mockVpoolKeeper.EXPECT().
				SwapQuoteForBase(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_REMOVE_FROM_POOL,
					/* quoteAmt */ tc.exchangedNotional,
					/* baseLimit */ sdk.ZeroDec(),
				).
				Return(tc.exchangedSize, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, common.DenomStable).
				Return(sdk.NewInt64Coin(common.DenomStable, 1_000))
			mocks.mockBankKeeper.EXPECT().
				SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidatorAddr,
					sdk.NewCoins(tc.expectedLiquidatorFee),
				).
				Return(nil)
			mocks.mockBankKeeper.EXPECT().
				SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(tc.expectedPerpEFFee),
				).
				Return(nil)

			t.Log("execute liquidation")
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee, feeToFund)

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.GetPosition(ctx, common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, traderAddr.String(), newPosition.TraderAddress)
			assert.EqualValues(t, common.PairBTCStable, newPosition.Pair)
			assert.EqualValues(t, tc.expectedPositionSize, newPosition.Size_)
			assert.EqualValues(t, tc.expectedPositionMargin, newPosition.Margin)
			assert.EqualValues(t, tc.expectedPositionOpenNotional, newPosition.OpenNotional)
			assert.True(t, newPosition.LastUpdateCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  common.PairBTCStable.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.exchangedNotional,
				ExchangedPositionSize: tc.exchangedSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               sdk.ZeroDec(),
				Margin:                sdk.NewCoin(common.DenomStable, tc.expectedPositionMargin.RoundInt()),
				PositionNotional:      tc.newPositionNotional.Sub(tc.exchangedNotional),
				PositionSize:          tc.initialPositionSize.Sub(tc.exchangedSize),
				UnrealizedPnl:         tc.expectedUnrealizedPnl,
				MarkPrice:             markPrice,
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestLiquidateIntoFullLiquidation(t *testing.T) {
	tests := []struct {
		name string

		initialPositionSize         sdk.Dec
		initialPositionMargin       sdk.Dec
		initialPositionOpenNotional sdk.Dec

		newPositionNotional sdk.Dec

		expectedLiquidatorFee sdk.Coin
		expectedPerpEFFee     sdk.Coin
	}{
		{
			name: "Full Liquidation - just under 1.25% margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(911), // just below 1.25% margin ratio

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 6), // 911 * 0.0125 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 5), // 11 - 6
		},
		{
			name: "Full Liquidation - at 0.625%",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.MustNewDecFromStr("905.6603774"), // at 0.625% margin ratio

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 6),
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 0),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := sample.AccAddress()
			liquidatorAddr := sample.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          common.PairBTCStable,
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			perpKeeper.SetPosition(ctx, common.PairBTCStable, traderAddr, &position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			})

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).Return(true).Times(2)
			mocks.mockVpoolKeeper.EXPECT().IsOverSpreadLimit(ctx, common.PairBTCStable).Return(false)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockVpoolKeeper.EXPECT().GetSpotPrice(ctx, common.PairBTCStable).Return(markPrice, nil)

			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.initialPositionSize,
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.initialPositionSize,
				).
				Return(tc.newPositionNotional, nil).Times(3)
			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					/* baseAmt */ tc.initialPositionSize,
					/* quoteLimit */ sdk.ZeroDec(),
				).
				Return(tc.newPositionNotional, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, common.DenomStable).
				Return(sdk.NewInt64Coin(common.DenomStable, 1_000))
			mocks.mockBankKeeper.EXPECT().
				SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidatorAddr,
					sdk.NewCoins(tc.expectedLiquidatorFee),
				).
				Return(nil)
			if tc.expectedPerpEFFee.Amount.IsPositive() {
				mocks.mockBankKeeper.EXPECT().
					SendCoinsFromModuleToModule(
						ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
						sdk.NewCoins(tc.expectedPerpEFFee),
					).
					Return(nil)
			}

			t.Log("execute liquidation")
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee.String(), feeToFund.String())

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.GetPosition(ctx, common.PairBTCStable, traderAddr)
			require.ErrorIs(t, err, types.ErrPositionNotFound)
			assert.Nil(t, newPosition)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  common.PairBTCStable.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.newPositionNotional,
				ExchangedPositionSize: tc.initialPositionSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               sdk.ZeroDec(),
				Margin:                sdk.NewCoin(common.DenomStable, sdk.ZeroInt()),
				PositionNotional:      sdk.ZeroDec(), // always zero
				PositionSize:          sdk.ZeroDec(), // always zero
				UnrealizedPnl:         sdk.ZeroDec(), // always zero
				MarkPrice:             markPrice,
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestLiquidateIntoFullLiquidationWithBadDebt(t *testing.T) {
	tests := []struct {
		name string

		initialPositionSize         sdk.Dec
		initialPositionMargin       sdk.Dec
		initialPositionOpenNotional sdk.Dec

		newPositionNotional sdk.Dec

		expectedLiquidatorFee sdk.Coin
		expectedPerpEFFee     sdk.Coin

		expectedPositionBadDebt    sdk.Dec
		expectedLiquidationBadDebt sdk.Dec
	}{
		{
			name: "Full Liquidation - at 0% margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(900), // at 0% margin ratio

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 6), // 900 * 0.0125 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 0), // no margin left for perp ef

			expectedPositionBadDebt:    sdk.ZeroDec(),
			expectedLiquidationBadDebt: sdk.MustNewDecFromStr("5.625"),
		},
		{
			name: "Full Liquidation - below 0% margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(899),

			expectedLiquidatorFee: sdk.NewInt64Coin(common.DenomStable, 6),
			expectedPerpEFFee:     sdk.NewInt64Coin(common.DenomStable, 0),

			expectedPositionBadDebt:    sdk.NewDec(1),
			expectedLiquidationBadDebt: sdk.MustNewDecFromStr("5.61875"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := sample.AccAddress()
			liquidatorAddr := sample.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          common.PairBTCStable,
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			perpKeeper.SetPosition(ctx, common.PairBTCStable, traderAddr, &position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			})

			t.Log("mock vpool keeper")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).Return(true).Times(2)
			mocks.mockVpoolKeeper.EXPECT().IsOverSpreadLimit(ctx, common.PairBTCStable).Return(false)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockVpoolKeeper.EXPECT().GetSpotPrice(ctx, common.PairBTCStable).Return(markPrice, nil)

			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.initialPositionSize,
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.initialPositionSize,
				).
				Return(tc.newPositionNotional, nil).Times(3)
			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.PairBTCStable,
					vpooltypes.Direction_ADD_TO_POOL,
					/* baseAmt */ tc.initialPositionSize,
					/* quoteLimit */ sdk.ZeroDec(),
				).
				Return(tc.newPositionNotional, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, common.DenomStable).
				Return(sdk.NewInt64Coin(common.DenomStable, 1_000))
			mocks.mockBankKeeper.EXPECT().
				SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidatorAddr,
					sdk.NewCoins(tc.expectedLiquidatorFee),
				).
				Return(nil)
			mocks.mockBankKeeper.EXPECT().
				SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount,
					sdk.NewCoins(
						sdk.NewCoin(
							common.DenomStable,
							tc.expectedLiquidationBadDebt.Add(tc.expectedPositionBadDebt).RoundInt(),
						),
					),
				).
				Return(nil)

			t.Log("execute liquidation")
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee.String(), feeToFund.String())

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.GetPosition(ctx, common.PairBTCStable, traderAddr)
			require.ErrorIs(t, err, types.ErrPositionNotFound)
			assert.Nil(t, newPosition)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  common.PairBTCStable.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.newPositionNotional,
				ExchangedPositionSize: tc.initialPositionSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               tc.expectedLiquidationBadDebt.Add(tc.expectedPositionBadDebt),
				Margin:                sdk.NewInt64Coin(common.DenomStable, 0),
				PositionNotional:      sdk.ZeroDec(), // always zero
				PositionSize:          sdk.ZeroDec(), // always zero
				UnrealizedPnl:         sdk.ZeroDec(), // always zero
				MarkPrice:             markPrice,
				BlockHeight:           ctx.BlockHeight(),
				BlockTimeMs:           ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestDistributeLiquidateRewards(t *testing.T) {
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
					types.LiquidateResp{
						BadDebt:                sdk.OneInt(),
						FeeToLiquidator:        sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             "",
					},
				)
				require.Error(t, err)
			},
		},
		{
			name: "vpool does not exist - error",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidator := sample.AccAddress()
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).Return(false)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{
						BadDebt:                sdk.OneInt(),
						FeeToLiquidator:        sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: common.PairBTCStable,
							},
						},
					},
				)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "healthy liquidation",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidator := sample.AccAddress()

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).Return(true)

				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))

				mocks.mockBankKeeper.
					EXPECT().GetBalance(ctx, authtypes.NewModuleAddress(types.VaultModuleAccount), "unusd").
					Return(sdk.NewCoin("unusd", sdk.NewInt(math.MaxInt64)))
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewCoin("unusd", sdk.OneInt())),
				).Return(nil)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidator,
					sdk.NewCoins(sdk.NewCoin("unusd", sdk.OneInt())),
				).Return(nil)

				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{
						BadDebt:                sdk.OneInt(),
						FeeToLiquidator:        sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: common.PairBTCStable,
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

func TestExecuteFullLiquidation(t *testing.T) {
	tests := []struct {
		name string

		liquidationFee      sdk.Dec
		initialPositionSize sdk.Dec
		initialMargin       sdk.Dec
		initialOpenNotional sdk.Dec

		// amount of quote obtained by trading <initialPositionSize> base
		baseAssetPriceInQuote sdk.Dec

		expectedLiquidationBadDebt     sdk.Int
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

			expectedLiquidationBadDebt:     sdk.ZeroInt(),
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

			expectedLiquidationBadDebt:     sdk.NewInt(5),
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

			expectedLiquidationBadDebt:     sdk.NewInt(22),
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

			expectedLiquidationBadDebt:     sdk.ZeroInt(),
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

			expectedLiquidationBadDebt:     sdk.NewInt(5),
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

			expectedLiquidationBadDebt:     sdk.NewInt(28),
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
					sdk.NewCoins(sdk.NewCoin("unusd", tc.expectedFundsToPerpEF)),
				).Return(nil)
			}
			if tc.expectedFundsToLiquidator.IsPositive() {
				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))
				mocks.mockBankKeeper.
					EXPECT().GetBalance(ctx, authtypes.NewModuleAddress(types.VaultModuleAccount), "unusd").
					Return(sdk.NewCoin("unusd", sdk.NewInt(math.MaxInt64)))
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, liquidatorAddr,
					sdk.NewCoins(sdk.NewCoin("unusd", tc.expectedFundsToLiquidator)),
				).Return(nil)
			}
			if tc.expectedLiquidationBadDebt.IsPositive() {
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount,
					sdk.NewCoins(sdk.NewCoin("unusd", tc.expectedLiquidationBadDebt)),
				)
			}

			t.Log("setup perp keeper params")
			newParams := types.DefaultParams()
			newParams.LiquidationFeeRatio = tc.liquidationFee
			perpKeeper.SetParams(ctx, newParams)
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(), // zero funding payment for this test case
				},
			})

			t.Log("mock vpool")
			mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, common.PairBTCStable).AnyTimes().Return(true)
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.PairBTCStable,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockVpoolKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					common.PairBTCStable,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
					/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
				).Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockVpoolKeeper.EXPECT().
				GetSpotPrice(ctx, common.PairBTCStable).
				Return(sdk.OneDec(), nil)

			t.Log("create and set the initial position")
			position := types.Position{
				TraderAddress:                       traderAddr.String(),
				Pair:                                common.PairBTCStable,
				Size_:                               tc.initialPositionSize,
				Margin:                              tc.initialMargin,
				OpenNotional:                        tc.initialOpenNotional,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         ctx.BlockHeight(),
			}
			perpKeeper.SetPosition(ctx, common.PairBTCStable, traderAddr, &position)

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
			assert.EqualValues(t, common.PairBTCStable, newPosition.Pair)
			assert.True(t, newPosition.Size_.IsZero())        // always zero
			assert.True(t, newPosition.Margin.IsZero())       // always zero
			assert.True(t, newPosition.OpenNotional.IsZero()) // always zero
			assert.True(t, newPosition.LastUpdateCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  common.PairBTCStable.String(),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
				ExchangedPositionSize: positionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       sdk.NewCoin(common.PairBTCStable.GetQuoteTokenDenom(), tc.expectedFundsToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(common.PairBTCStable.GetQuoteTokenDenom(), tc.expectedFundsToPerpEF),
				BadDebt:               tc.expectedLiquidationBadDebt.ToDec(),
				Margin:                sdk.NewCoin(common.PairBTCStable.GetQuoteTokenDenom(), newPosition.Margin.RoundInt()),
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
