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
				pair := common.TokenPair("xxx:yyy")
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

func TestExecuteFullLiquidation_Unit(t *testing.T) {
	testCases := []struct {
		name           string
		position       *types.Position
		liquidationFee sdk.Dec
	}{
		{
			name: "liquidateEmptyPositionBUY",
			position: &types.Position{
				Margin:       sdk.NewDec(100),
				Size_:        sdk.NewDec(1_000),
				OpenNotional: sdk.NewDec(1_000)},
			liquidationFee: sdk.MustNewDecFromStr("0.1"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// k, mocks, ctx := getKeeper(t)
			k, _, ctx := getKeeper(t)

			trader := sample.AccAddress()
			pair := common.TokenPair("xxx:yyy")
			tc.position.Address = trader.String()
			tc.position.Pair = pair.String()

			t.Log("Set vpool defined by pair on VpoolKeeper")
			// mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

			t.Log("Setup params and pair metadata")
			params := types.DefaultParams()
			k.SetParams(ctx, types.NewParams(
				params.Stopped,
				params.MaintenanceMarginRatio,
				params.GetTollRatioAsDec(),
				params.GetSpreadRatioAsDec(),
				tc.liquidationFee,
				params.GetPartialLiquidationRatioAsDec(),
			))
			k.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       pair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("Initialize the test case position")
			k.SetPosition(ctx, pair, trader.String(), tc.position)
			_, err := k.GetPosition(ctx, pair, trader.String())
			require.NoError(t, err)

			t.Log("Run 'executeFullLiquidation'")
			// liquidator := sample.AccAddress()
			// err = k.ExecuteFullLiquidation(ctx, liquidator, tc.position)
			// require.Error(t, err)
		})
	}
}
