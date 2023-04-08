package keeper

import (
	"math"
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
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

			newPositionNotional: sdk.NewDec(959),                 // just below 6.25% margin ratio
			exchangedSize:       sdk.MustNewDecFromStr("0.25"),   // 1 * 0.25
			exchangedNotional:   sdk.MustNewDecFromStr("239.75"), // 959 * 0.25

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 3), // 959 * 0.25 * 0.025 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 3), // 959 * 0.25 * 0.025 / 2

			expectedPositionSize:         sdk.MustNewDecFromStr("0.75"),     // 1 - 0.25
			expectedUnrealizedPnl:        sdk.MustNewDecFromStr("-30.75"),   // -41 * 0.75
			expectedPositionMargin:       sdk.MustNewDecFromStr("83.75625"), // 100 - 20.5 - 959 * 0.25 * 0.025
			expectedPositionOpenNotional: sdk.NewDec(750),
		},
		{
			name: "Partial Liquidation - just above full liquidation",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.MustNewDecFromStr("923.0769231"),  // at 2.5% margin ratio
			exchangedSize:       sdk.MustNewDecFromStr("0.25"),         // 1 * 0.5
			exchangedNotional:   sdk.MustNewDecFromStr("230.76923078"), // 923.0769231 * 0.5

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 3), // 923.0769231 * 0.25 * 0.025 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 3), // 923.0769231 * 0.25 * 0.025 / 2

			expectedPositionSize:         sdk.MustNewDecFromStr("0.75"),          // 1 - 0.25
			expectedUnrealizedPnl:        sdk.MustNewDecFromStr("-57.692307675"), // -76.92307692 * 0.75
			expectedPositionMargin:       sdk.MustNewDecFromStr("75.0000000055"), // 100 - 19.23076923 - 923.0769231 * 0.25 * 0.025
			expectedPositionOpenNotional: sdk.MustNewDecFromStr("749.999999995"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := testutilevents.AccAddress()
			liquidatorAddr := testutilevents.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			SetPosition(perpKeeper, ctx, position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("mock vpool keeper")
			vpool := perpammtypes.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockPerpAmmKeeper.EXPECT().
				GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Times(2).
				Return(vpool, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true).Times(1)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.MustNewDecFromStr("0.0625"), nil)

			mocks.mockPerpAmmKeeper.EXPECT().IsOverSpreadLimit(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(false, nil)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockPerpAmmKeeper.EXPECT().GetMarkPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(markPrice, nil)

			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					perpammtypes.Direction_LONG,
					sdk.OneDec(),
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					perpammtypes.Direction_LONG,
					sdk.OneDec(),
				).
				Return(tc.newPositionNotional, nil).Times(3)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					perpammtypes.Direction_LONG,
					tc.exchangedSize,
				).
				Return(tc.exchangedNotional, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				SwapQuoteForBase(
					ctx,
					vpool,
					perpammtypes.Direction_SHORT,
					/* quoteAmt */ tc.exchangedNotional,
					/* baseLimit */ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ true,
				).
				Return(vpool, tc.exchangedSize, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, denoms.NUSD).
				Return(sdk.NewInt64Coin(denoms.NUSD, 1_000))
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
			setLiquidator(ctx, perpKeeper, liquidatorAddr)
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee, feeToFund)

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.NoError(t, err)
			assert.EqualValues(t, traderAddr.String(), newPosition.TraderAddress)
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), newPosition.Pair)
			assert.EqualValues(t, tc.expectedPositionSize, newPosition.Size_)
			assert.EqualValues(t, tc.expectedPositionMargin, newPosition.Margin)
			assert.EqualValues(t, tc.expectedPositionOpenNotional, newPosition.OpenNotional)
			assert.True(t, newPosition.LatestCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.exchangedNotional,
				ExchangedPositionSize: tc.exchangedSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				Margin:                sdk.NewCoin(denoms.NUSD, tc.expectedPositionMargin.RoundInt()),
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
			name: "Full Liquidation - just under 2.5% margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(923), // just below 2.5% margin ratio

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 12), // 923 * 0.025 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 11), // 23 - 12
		},
		{
			name: "Full Liquidation - at 1.25%",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.MustNewDecFromStr("911.3924051"), // at 1.25% margin ratio

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 11),
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 0),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := testutilevents.AccAddress()
			liquidatorAddr := testutilevents.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			SetPosition(perpKeeper, ctx, position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("mock vpool keeper")
			vpool := perpammtypes.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockPerpAmmKeeper.EXPECT().
				GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Times(2).
				Return(vpool, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.MustNewDecFromStr("0.0625"), nil)
			mocks.mockPerpAmmKeeper.EXPECT().IsOverSpreadLimit(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(false, nil)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockPerpAmmKeeper.EXPECT().GetMarkPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(markPrice, nil)

			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					perpammtypes.Direction_LONG,
					tc.initialPositionSize,
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					perpammtypes.Direction_LONG,
					tc.initialPositionSize,
				).
				Return(tc.newPositionNotional, nil).Times(3)
			mocks.mockPerpAmmKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					vpool,
					perpammtypes.Direction_LONG,
					/* baseAmt */ tc.initialPositionSize,
					/* quoteLimit */ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ true,
				).
				Return(vpool, tc.newPositionNotional, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, denoms.NUSD).
				Return(sdk.NewInt64Coin(denoms.NUSD, 1_000))
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
			setLiquidator(ctx, perpKeeper, liquidatorAddr)
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee.String(), feeToFund.String())

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.ErrorIs(t, err, collections.ErrNotFound)
			assert.Empty(t, newPosition)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.newPositionNotional,
				ExchangedPositionSize: tc.initialPositionSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				Margin:                sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
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

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 11), // 900 * 0.025 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 0),  // no margin left for perp ef

			expectedPositionBadDebt:    sdk.ZeroDec(),
			expectedLiquidationBadDebt: sdk.MustNewDecFromStr("11.25"),
		},
		{
			name: "Full Liquidation - below 0% margin ratio",

			initialPositionSize:         sdk.OneDec(),
			initialPositionMargin:       sdk.NewDec(100),
			initialPositionOpenNotional: sdk.NewDec(1000),

			newPositionNotional: sdk.NewDec(899),

			expectedLiquidatorFee: sdk.NewInt64Coin(denoms.NUSD, 11), // 899 * 0.025 / 2
			expectedPerpEFFee:     sdk.NewInt64Coin(denoms.NUSD, 0),

			expectedPositionBadDebt:    sdk.NewDec(1),
			expectedLiquidationBadDebt: sdk.MustNewDecFromStr("11.2375"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			traderAddr := testutilevents.AccAddress()
			liquidatorAddr := testutilevents.AccAddress()
			vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)

			t.Log("set position")
			position := types.Position{
				TraderAddress: traderAddr.String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         tc.initialPositionSize,
				Margin:        tc.initialPositionMargin,
				OpenNotional:  tc.initialPositionOpenNotional,
			}
			SetPosition(perpKeeper, ctx, position)

			t.Log("set params")
			params := types.DefaultParams()
			perpKeeper.SetParams(ctx, params)

			t.Log("set pair metadata")
			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("mock vpool keeper")
			vpool := perpammtypes.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockPerpAmmKeeper.EXPECT().
				GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Times(2).
				Return(vpool, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetMaintenanceMarginRatio(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.MustNewDecFromStr("0.0625"), nil)
			mocks.mockPerpAmmKeeper.EXPECT().IsOverSpreadLimit(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(false, nil)
			markPrice := tc.newPositionNotional.Quo(tc.initialPositionSize)
			mocks.mockPerpAmmKeeper.EXPECT().GetMarkPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(markPrice, nil)

			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					perpammtypes.Direction_LONG,
					tc.initialPositionSize,
					15*time.Minute,
				).
				Return(tc.newPositionNotional, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					perpammtypes.Direction_LONG,
					tc.initialPositionSize,
				).
				Return(tc.newPositionNotional, nil).Times(3)
			mocks.mockPerpAmmKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					vpool,
					perpammtypes.Direction_LONG,
					/* baseAmt */ tc.initialPositionSize,
					/* quoteLimit */ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ true,
				).
				Return(vpool, tc.newPositionNotional, nil)

			t.Log("mock account keeper")
			mocks.mockAccountKeeper.EXPECT().
				GetModuleAddress(types.VaultModuleAccount).
				Return(vaultAddr)

			t.Log("mock bank keeper")
			mocks.mockBankKeeper.EXPECT().
				GetBalance(ctx, vaultAddr, denoms.NUSD).
				Return(sdk.NewInt64Coin(denoms.NUSD, 1_000))
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
							denoms.NUSD,
							tc.expectedLiquidationBadDebt.Add(tc.expectedPositionBadDebt).RoundInt(),
						),
					),
				).
				Return(nil)

			t.Log("execute liquidation")
			setLiquidator(ctx, perpKeeper, liquidatorAddr)
			feeToLiquidator, feeToFund, err := perpKeeper.Liquidate(ctx, liquidatorAddr, asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedLiquidatorFee, feeToLiquidator)
			assert.EqualValues(t, tc.expectedPerpEFFee.String(), feeToFund.String())

			t.Log("assert new position and event")
			newPosition, err := perpKeeper.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.ErrorIs(t, err, collections.ErrNotFound)
			assert.Empty(t, newPosition)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  tc.newPositionNotional,
				ExchangedPositionSize: tc.initialPositionSize.Neg(),
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       tc.expectedLiquidatorFee,
				FeeToEcosystemFund:    tc.expectedPerpEFFee,
				BadDebt:               sdk.NewCoin(denoms.NUSD, tc.expectedLiquidationBadDebt.Add(tc.expectedPositionBadDebt).RoundInt()),
				Margin:                sdk.NewInt64Coin(denoms.NUSD, 0),
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
				liquidator := testutilevents.AccAddress()
				mocks.mockPerpAmmKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(false)

				err := perpKeeper.distributeLiquidateRewards(ctx,
					types.LiquidateResp{
						BadDebt:                sdk.OneInt(),
						FeeToLiquidator:        sdk.OneInt(),
						FeeToPerpEcosystemFund: sdk.OneInt(),
						Liquidator:             liquidator.String(),
						PositionResp: &types.PositionResp{
							Position: &types.Position{
								Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
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
				liquidator := testutilevents.AccAddress()

				mocks.mockPerpAmmKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(true)

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
								Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
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

func TestKeeper_ExecuteFullLiquidation(t *testing.T) {
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
			liquidatorAddr := testutilevents.AccAddress()
			traderAddr := testutilevents.AccAddress()
			baseAssetDirection := perpammtypes.Direction_LONG
			if tc.initialPositionSize.IsNegative() {
				baseAssetDirection = perpammtypes.Direction_SHORT
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
			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("mock vpool")
			vpool := perpammtypes.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockPerpAmmKeeper.EXPECT().GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).Return(vpool, nil)
			mocks.mockPerpAmmKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).AnyTimes().Return(true)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				SwapBaseForQuote(
					ctx,
					vpool,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
					/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
					/* skipFluctuationLimitCheck */ true,
				).Return(vpool /*quoteAssetAmount=*/, tc.baseAssetPriceInQuote, nil)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetMarkPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.OneDec(), nil)

			t.Log("create and set the initial position")
			position := types.Position{
				TraderAddress:                   traderAddr.String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           tc.initialPositionSize,
				Margin:                          tc.initialMargin,
				OpenNotional:                    tc.initialOpenNotional,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     ctx.BlockHeight(),
			}
			SetPosition(perpKeeper, ctx, position)

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
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), newPosition.Pair)
			assert.True(t, newPosition.Size_.IsZero())        // always zero
			assert.True(t, newPosition.Margin.IsZero())       // always zero
			assert.True(t, newPosition.OpenNotional.IsZero()) // always zero
			assert.True(t, newPosition.LatestCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
				ExchangedPositionSize: positionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), tc.expectedFundsToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), tc.expectedFundsToPerpEF),
				BadDebt:               sdk.NewCoin(denoms.NUSD, tc.expectedLiquidationBadDebt),
				Margin:                sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), newPosition.Margin.RoundInt()),
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

func TestKeeper_ExecutePartialLiquidation(t *testing.T) {
	tests := []struct {
		name string

		liquidationFee          sdk.Dec
		partialLiquidationRatio sdk.Dec
		initialPositionSize     sdk.Dec
		initialMargin           sdk.Dec
		initialOpenNotional     sdk.Dec

		// amount of quote obtained by trading <initialPositionSize> base
		baseAssetPriceInQuote sdk.Dec

		expectedFundsToPerpEF          sdk.Int
		expectedFundsToLiquidator      sdk.Int
		expectedExchangedNotionalValue sdk.Dec
		expectedMarginToVault          sdk.Dec
		expectedPositionRealizedPnl    sdk.Dec
		expectedPositionMargin         sdk.Dec
	}{
		{
			/*
				- long position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)

				- no funding payment

				- liquidation fee ratio is 0.1
				- partial liquidation ratio is 0.2
				- notional exchanged is 20 NUSD
				- liquidator gets 20 NUSD * 0.1 / 2 =  1 NUSD
				- ecosystem fund gets remaining = 1 NUSD
			*/
			name: "remaining margin more than liquidation fee",

			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.2"),
			initialPositionSize:     sdk.NewDec(100),
			initialMargin:           sdk.NewDec(10),
			initialOpenNotional:     sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedFundsToPerpEF:          sdk.NewInt(1),
			expectedFundsToLiquidator:      sdk.NewInt(1),
			expectedExchangedNotionalValue: sdk.NewDec(20),
			expectedMarginToVault:          sdk.NewDec(0),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionMargin:         sdk.NewDec(8),
		},
		{
			/*
				- short position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)

				- no funding payment

				- liquidation fee ratio is 0.1
				- partial liquidation ratio is 0.2
				- notional exchanged is 20 NUSD
				- liquidator gets 20 NUSD * 0.1 / 2 =  1 NUSD
				- ecosystem fund gets remaining = 1 NUSD
			*/
			name: "remaining margin more than liquidation fee",

			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.2"),
			initialPositionSize:     sdk.NewDec(-100),
			initialMargin:           sdk.NewDec(10),
			initialOpenNotional:     sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(100), // no change in price

			expectedFundsToPerpEF:          sdk.NewInt(1),
			expectedFundsToLiquidator:      sdk.NewInt(1),
			expectedExchangedNotionalValue: sdk.NewDec(20),
			expectedMarginToVault:          sdk.ZeroDec(),
			expectedPositionRealizedPnl:    sdk.ZeroDec(),
			expectedPositionMargin:         sdk.NewDec(8),
		},
		{
			/*
				- long position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)
				- BTC drops in price (1 BTC = 0.8 NUSD)
				- position notional of 80 NUSD
				- unrealized PnL of -20 NUSD

				- no funding payment

				- liquidation fee ratio is 0.1
				- partial liquidation ration is 0.2
				- notional exchanged is 16 NUSD
				- liquidator gets 16 NUSD * 0.1 / 2 = 1 NUSD
				- ecosystem fund gets 1 NUSD
			*/
			name: "position has + margin",

			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.2"),
			initialPositionSize:     sdk.NewDec(100),
			initialMargin:           sdk.NewDec(10),
			initialOpenNotional:     sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(80), // price dropped

			expectedFundsToPerpEF:          sdk.NewInt(1),
			expectedFundsToLiquidator:      sdk.NewInt(1),
			expectedExchangedNotionalValue: sdk.NewDec(16),
			expectedMarginToVault:          sdk.ZeroDec(),
			expectedPositionRealizedPnl:    sdk.MustNewDecFromStr("-3.2"),
			expectedPositionMargin:         sdk.MustNewDecFromStr("5.2"),
		},
		{
			/*
				- short position
				- open margin 10 NUSD, 10x leverage
				- open notional and position notional of 100 NUSD
				- position size 100 BTC (1 BTC = 1 NUSD)
				- BTC increases in price (1 BTC = 1.2 NUSD)
				- position notional of 120 NUSD
				- unrealized PnL of -20 NUSD,

				- no funding payment

				- liquidation fee ratio is 0.1
				- partial liquidation ratio is 0.2
				- notional exchanged is 120*0.2 (partial liquidation ratio) = 24 NUSD
				- liquidator gets 24 NUSD * 0.1 / 2 = 1 NUSD
				- ecosystem fund gets 1 NUSD
			*/
			name: "position has + margin",

			liquidationFee:          sdk.MustNewDecFromStr("0.1"),
			partialLiquidationRatio: sdk.MustNewDecFromStr("0.2"),
			initialPositionSize:     sdk.NewDec(-100),
			initialMargin:           sdk.NewDec(10),
			initialOpenNotional:     sdk.NewDec(100),

			baseAssetPriceInQuote: sdk.NewDec(120), // price increased

			expectedFundsToPerpEF:          sdk.NewInt(1),
			expectedFundsToLiquidator:      sdk.NewInt(1),
			expectedExchangedNotionalValue: sdk.NewDec(24),
			expectedMarginToVault:          sdk.ZeroDec(),
			expectedPositionRealizedPnl:    sdk.MustNewDecFromStr("-4.8"),
			expectedPositionMargin:         sdk.MustNewDecFromStr("2.8"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("setup variables")
			perpKeeper, mocks, ctx := getKeeper(t)
			liquidatorAddr := testutilevents.AccAddress()
			traderAddr := testutilevents.AccAddress()
			baseAssetDirection := perpammtypes.Direction_LONG
			if tc.initialPositionSize.IsNegative() {
				baseAssetDirection = perpammtypes.Direction_SHORT
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

			t.Log("setup perp keeper params")
			newParams := types.DefaultParams()
			newParams.LiquidationFeeRatio = tc.liquidationFee
			newParams.PartialLiquidationRatio = tc.partialLiquidationRatio
			perpKeeper.SetParams(ctx, newParams)
			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("mock vpool")
			vpool := perpammtypes.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockPerpAmmKeeper.EXPECT().
				GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(vpool, nil)
			mocks.mockPerpAmmKeeper.EXPECT().ExistsPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).AnyTimes().Return(true)
			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Mul(tc.partialLiquidationRatio),
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio), nil)

			mocks.mockPerpAmmKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					baseAssetDirection,
					/*baseAssetAmount=*/ tc.initialPositionSize.Abs(),
				).
				Return( /*quoteAssetAmount=*/ tc.baseAssetPriceInQuote, nil)

			if tc.initialPositionSize.IsNegative() {
				mocks.mockPerpAmmKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						vpool,
						perpammtypes.Direction_LONG,
						/*baseAssetAmount=*/ tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio),
						/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
						/* skipFluctuationLimitCheck */ true,
					).Return(vpool /*quoteAssetAmount=*/, tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio), nil)
			} else {
				mocks.mockPerpAmmKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						vpool,
						perpammtypes.Direction_SHORT,
						/*baseAssetAmount=*/ tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio),
						/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
						/* skipFluctuationLimitCheck */ true,
					).Return(vpool /*quoteAssetAmount=*/, tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio), nil)
			}

			mocks.mockPerpAmmKeeper.EXPECT().
				GetMarkPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD)).
				Return(sdk.OneDec(), nil)

			t.Log("create and set the initial position")
			position := types.Position{
				TraderAddress:                   traderAddr.String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           tc.initialPositionSize,
				Margin:                          tc.initialMargin,
				OpenNotional:                    tc.initialOpenNotional,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     ctx.BlockHeight(),
			}
			SetPosition(perpKeeper, ctx, position)

			t.Log("execute partial liquidation")
			liquidationResp, err := perpKeeper.ExecutePartialLiquidation(
				ctx, liquidatorAddr, &position)
			require.NoError(t, err)

			t.Log("assert liquidation response fields")
			assert.True(t, liquidationResp.BadDebt.IsZero())
			assert.EqualValues(t, tc.expectedFundsToLiquidator, liquidationResp.FeeToLiquidator)
			assert.EqualValues(t, tc.expectedFundsToPerpEF, liquidationResp.FeeToPerpEcosystemFund)
			assert.EqualValues(t, liquidatorAddr.String(), liquidationResp.Liquidator)

			t.Log("assert position response fields")
			positionResp := liquidationResp.PositionResp
			assert.EqualValues(t,
				tc.expectedExchangedNotionalValue,
				positionResp.ExchangedNotionalValue) // amount of quote exchanged
			// Initial position size that is liquidated to be is sold back to to vpool
			if tc.initialPositionSize.IsNegative() {
				assert.EqualValues(t, tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio), positionResp.ExchangedPositionSize)
			} else {
				assert.EqualValues(t, tc.baseAssetPriceInQuote.Mul(tc.partialLiquidationRatio).Neg(), positionResp.ExchangedPositionSize)
			}
			// ( oldMargin + unrealizedPnL - fundingPayment ) * -1
			assert.True(t, tc.expectedMarginToVault.IsZero())
			//assert.EqualValues(t, tc.expectedPositionBadDebt, positionResp.BadDebt)
			assert.EqualValues(t, tc.expectedPositionRealizedPnl, positionResp.RealizedPnl)
			assert.True(t, positionResp.FundingPayment.IsZero())

			t.Log("assert new position fields")
			newPosition := positionResp.Position
			assert.EqualValues(t, traderAddr.String(), newPosition.TraderAddress)
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), newPosition.Pair)
			assert.True(t, newPosition.LatestCumulativePremiumFraction.IsZero())
			assert.EqualValues(t, ctx.BlockHeight(), newPosition.BlockNumber)
			assert.EqualValues(t, tc.expectedPositionMargin, newPosition.Margin)

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionLiquidatedEvent{
				Pair:                  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:         traderAddr.String(),
				ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
				ExchangedPositionSize: positionResp.ExchangedPositionSize,
				LiquidatorAddress:     liquidatorAddr.String(),
				FeeToLiquidator:       sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), tc.expectedFundsToLiquidator),
				FeeToEcosystemFund:    sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), tc.expectedFundsToPerpEF),
				BadDebt:               sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), sdk.ZeroInt()),
				Margin:                sdk.NewCoin(asset.Registry.Pair(denoms.BTC, denoms.NUSD).QuoteDenom(), newPosition.Margin.RoundInt()),
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

func setLiquidator(ctx sdk.Context, k Keeper, addr sdk.AccAddress) {
	p := k.GetParams(ctx)
	p.WhitelistedLiquidators = []string{addr.String()}
	k.SetParams(ctx, p)
}
