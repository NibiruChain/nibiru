package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func setUp(t *testing.T) (perpKeeper Keeper, mocks mockedDependencies, ctx sdk.Context, traderAddr sdk.AccAddress, liquidatorAddr sdk.AccAddress) {
	perpKeeper, mocks, ctx = getKeeper(t)
	perpKeeper.SetParams(ctx, types.DefaultParams())

	traderAddr = sample.AccAddress()
	liquidatorAddr = sample.AccAddress()

	return
}

func TestLiquidate_Unit(t *testing.T) {
	testcases := []struct {
		name string
		test func()
	}{
		{
			name: "long position; negative pnl; margin below maintenance",
			test: func() {
				/*
					open long position 10 BTC:USD at 20k BTC:USD with 15k usd margin
				*/

				perpKeeper, mocks, ctx, traderAddr, liquidatorAddr := setUp(t)

				params := types.DefaultParams()
				perpKeeper.SetParams(ctx, types.NewParams(
					params.Stopped,
					sdk.MustNewDecFromStr("1"),
					params.GetTollRatioAsDec(),
					params.GetSpreadRatioAsDec(),
					params.GetLiquidationFeeAsDec(),
					params.GetPartialLiquidationRatioAsDec(),
				))

				pairStr := "BTC:NUSD"
				pair := common.TokenPair(pairStr)

				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair:                       pair.String(),
					CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
				})

				t.Log("Mocking price of vpool")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10), // buy
					).AnyTimes().
					Return(sdk.NewDec(20_000), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20_000), nil)

				mocks.mockVpoolKeeper.EXPECT().
					IsOverSpreadLimit(
						ctx,
						common.TokenPair(pair),
					).
					Return(false)

				t.Log("Opening the position")
				toLiquidatePosition := &types.Position{
					Address:      traderAddr.String(),
					Pair:         pairStr,
					Size_:        sdk.NewDec(10),
					OpenNotional: sdk.NewDec(200_000),
					Margin:       sdk.NewDec(15_000),
				}
				perpKeeper.SetPosition(ctx, pair, traderAddr.String(), toLiquidatePosition)

				t.Log("After the position is opened, the vpool price changes")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(1000),
					).AnyTimes().
					Return(sdk.NewDec(20_000), nil)

				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						/* pair */ common.TokenPair(pair),
						/* dir */ vpooltypes.Direction_ADD_TO_POOL,
						/* abs */ sdk.NewDec(10),
						/* limit */ sdk.ZeroDec(),
					).
					Return(sdk.NewDec(20_000), nil)

				t.Log("Successful liquidation will send funds to the liquidator")
				mocks.mockBankKeeper.EXPECT().
					SendCoinsFromModuleToAccount(
						ctx,
						types.PerpEFModuleAccount,
						liquidatorAddr,
						sdk.NewCoins(sdk.NewCoin("NUSD", sdk.NewInt(125))),
					).
					Return(nil)

				t.Log("Liquidating the position - should pass")
				err := perpKeeper.Liquidate(ctx, pair, traderAddr, liquidatorAddr)
				require.NoError(t, err)

				t.Log("Check that correct events emitted")
				expectedEvents := []sdk.Event{
					events.NewPositionLiquidateEvent(
						/* vpool */ pair.String(),
						/* owner */ traderAddr,
						/* notional */ sdk.NewDec(20_000),
						/* vsize */ sdk.NewDec(-10),
						/* liquidator */ liquidatorAddr,
						/* liquidationFee */ sdk.NewInt(125),
						/* badDebt */ sdk.NewDec(165_125),
					),
				}
				for _, event := range expectedEvents {
					assert.Contains(t, ctx.EventManager().Events(), event)
				}

			},
		},
		// {
		// 	name: "long position; margin ok",
		// 	test: func() {
		// 		perpKeeper, mocks, ctx, traderAddr, liquidatorAddr := setUp(t)

		// 		pairStr := "BTC:NUSD"
		// 		pair := common.TokenPair(pairStr)

		// 		perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
		// 			Pair:                       pair.String(),
		// 			CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
		// 		})

		// 		t.Log("Mocking price of vpool")
		// 		mocks.mockVpoolKeeper.EXPECT().
		// 			GetBaseAssetPrice(
		// 				ctx,
		// 				common.TokenPair(pair),
		// 				vpooltypes.Direction_ADD_TO_POOL,
		// 				sdk.NewDec(10),
		// 			).AnyTimes().
		// 			Return(sdk.NewDec(20), nil)

		// 		mocks.mockVpoolKeeper.EXPECT().
		// 			GetBaseAssetTWAP(
		// 				ctx,
		// 				common.TokenPair(pair),
		// 				vpooltypes.Direction_ADD_TO_POOL,
		// 				sdk.NewDec(10),
		// 				15*time.Minute,
		// 			).
		// 			Return(sdk.NewDec(20), nil)

		// 		mocks.mockVpoolKeeper.EXPECT().
		// 			IsOverSpreadLimit(
		// 				ctx,
		// 				common.TokenPair(pair),
		// 			).
		// 			Return(false)

		// 		t.Log("Opening the position")
		// 		toLiquidatePosition := &types.Position{
		// 			Address:      traderAddr.String(),
		// 			Pair:         pairStr,
		// 			Size_:        sdk.NewDec(10),
		// 			OpenNotional: sdk.NewDec(10),
		// 			Margin:       sdk.NewDec(1),
		// 		}
		// 		perpKeeper.SetPosition(ctx, pair, traderAddr.String(), toLiquidatePosition)

		// 		t.Log("After the position is opened, the vpool price changes")
		// 		mocks.mockVpoolKeeper.EXPECT().
		// 			GetBaseAssetPrice(
		// 				ctx,
		// 				common.TokenPair(pair),
		// 				vpooltypes.Direction_ADD_TO_POOL,
		// 				sdk.NewDec(5),
		// 			).AnyTimes().
		// 			Return(sdk.NewDec(20), nil)

		// 		t.Log("Liquidating the position")
		// 		err := perpKeeper.Liquidate(ctx, pair, traderAddr, liquidatorAddr)
		// 		require.ErrorIs(t, types.MarginHighEnough, err)
		// 	},
		// },

	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
