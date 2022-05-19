package keeper

import (
	"testing"
	"time"

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
					LiquidateResp{})
				require.Error(t, err)
				require.ErrorContains(t, err, "must not have nil fields")
			},
		},
		{
			name: "invalid liquidator - error",
			test: func() {
				perpKeeper, _, ctx := getKeeper(t)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
						FeeToPerpEcosystemFund: sdk.OneDec(),
						Liquidator:             sdk.AccAddress{},
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
					LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
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
				pair := common.TokenPair("xxx:yyy")
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(false)
				err := perpKeeper.distributeLiquidateRewards(ctx,
					LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
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
				pair := common.TokenPair("xxx:yyy")

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
					sdk.NewCoins(sdk.NewCoin("yyy", sdk.OneInt())),
				).Return(nil)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.PerpEFModuleAccount, liquidator,
					sdk.NewCoins(sdk.NewCoin("yyy", sdk.OneInt())),
				).Return(nil)

				err := perpKeeper.distributeLiquidateRewards(ctx,
					LiquidateResp{BadDebt: sdk.OneDec(), FeeToLiquidator: sdk.OneDec(),
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
						/* coin */ sdk.NewCoin("yyy", sdk.OneInt()),
						/* from */ vaultAddr.String(),
						/* to */ perpEFAddr.String(),
					),
					events.NewTransferEvent(
						/* coin */ sdk.NewCoin("yyy", sdk.OneInt()),
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

				t.Log("Liquidate the position - should pass")
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

				t.Log("Mocking the getting of addresses")
				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))

				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.PerpEFModuleAccount).
					Return(authtypes.NewModuleAddress(types.PerpEFModuleAccount))

				err := perpKeeper.Liquidate(ctx, liquidatorAddr, toLiquidatePosition)
				require.NoError(t, err)

				// t.Log("Check that correct events emitted")
				// expectedEvents := []sdk.Event{
				// 	events.NewPositionLiquidateEvent(
				// 		/* vpool */ pair.String(),
				// 		/* owner */ traderAddr,
				// 		/* notional */ sdk.NewDec(20_000),
				// 		/* vsize */ sdk.NewDec(-10),
				// 		/* liquidator */ liquidatorAddr,
				// 		/* liquidationFee */ sdk.NewInt(125),
				// 		/* badDebt */ sdk.NewDec(165_125),
				// 	),
				// }
				// for _, event := range expectedEvents {
				// 	assert.Contains(t, ctx.EventManager().Events(), event)
				// }
			},
		},
		{
			name: "long position; margin ok",
			test: func() {
				perpKeeper, mocks, ctx, traderAddr, liquidatorAddr := setUp(t)

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
						sdk.NewDec(10),
					).AnyTimes().
					Return(sdk.NewDec(20), nil)

				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)

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
					OpenNotional: sdk.NewDec(10),
					Margin:       sdk.NewDec(1),
				}
				perpKeeper.SetPosition(ctx, pair, traderAddr.String(), toLiquidatePosition)

				t.Log("After the position is opened, the vpool price changes")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						ctx,
						common.TokenPair(pair),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(5),
					).AnyTimes().
					Return(sdk.NewDec(20), nil)

				t.Log("Liquidate the position")
				//mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

				err := perpKeeper.Liquidate(ctx, liquidatorAddr, toLiquidatePosition)
				require.ErrorIs(t, types.ErrMarginHighEnough, err)
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
