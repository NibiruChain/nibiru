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
		test func()
	}{
		{
			/*
				- long position
				- remaining margin more than liquidation fee
				- no bad debt
				- no funding payment

				- liquidation fee is 0.1
				- notional exchanged is 100 NUSD
				- liquidator gets 100 NUSD * 0.1 / 2 = 5 NUSD
				- ecosystem fund gets remaining = 5 NUSD
			*/
			name: "remaining margin more than liquidation fee",
			test: func() {
				t.Log("setup variables")
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidatorAddr := sample.AccAddress()
				traderAddr := sample.AccAddress()
				pair := common.TokenPair("BTC:NUSD")

				t.Log("mock account keeper")
				vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				perpEFAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.VaultModuleAccount).
					Return(vaultAddr)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.PerpEFModuleAccount).
					Return(perpEFAddr)

				t.Log("mock bank keeper")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.VaultModuleAccount, types.PerpEFModuleAccount,
					sdk.NewCoins(sdk.NewInt64Coin("NUSD", 5)),
				).Return(nil)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.PerpEFModuleAccount, liquidatorAddr,
					sdk.NewCoins(sdk.NewInt64Coin("NUSD", 5)),
				).Return(nil)

				t.Log("setup perp keeper params")
				newParams := types.DefaultParams()
				newParams.LiquidationFee = 100_000 // liquidation fee ratio is 0.1
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
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)
				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						pair,
						/*baseAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("create and set the initial position")
				position := types.Position{
					Address:                             traderAddr.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(100),
					Margin:                              sdk.NewDec(10),
					OpenNotional:                        sdk.NewDec(100),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         ctx.BlockHeight(),
				}
				perpKeeper.SetPosition(ctx, pair, traderAddr.String(), &position)

				t.Log("execute full liquidation")
				liquidationResp, err := perpKeeper.ExecuteFullLiquidation(ctx, liquidatorAddr, &position)
				require.NoError(t, err)

				t.Log("assert liquidation response fields")
				assert.EqualValues(t, sdk.ZeroDec(), liquidationResp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(5), liquidationResp.FeeToLiquidator)
				assert.EqualValues(t, sdk.NewDec(5), liquidationResp.FeeToPerpEcosystemFund)
				assert.EqualValues(t, liquidatorAddr, liquidationResp.Liquidator)

				t.Log("assert position response fields")
				positionResp := liquidationResp.PositionResp
				assert.EqualValues(t, sdk.NewDec(100), positionResp.ExchangedQuoteAssetAmount) // amount of quote obtained
				assert.EqualValues(t, sdk.NewDec(-100), positionResp.ExchangedPositionSize)    // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(-10), positionResp.MarginToVault)             // ( 10(oldMargin) + 0(unrealzedPnL) - 0(fundingPayment) ) * -1
				assert.True(t, positionResp.BadDebt.IsZero())
				assert.True(t, positionResp.FundingPayment.IsZero())
				assert.True(t, positionResp.RealizedPnl.IsZero())
				assert.True(t, positionResp.UnrealizedPnlAfter.IsZero())

				t.Log("assert current position fields")
				currentPosition := positionResp.Position
				assert.EqualValues(t, traderAddr.String(), currentPosition.Address)
				assert.EqualValues(t, pair.String(), currentPosition.Pair)
				assert.True(t, currentPosition.Size_.IsZero())        // always zero
				assert.True(t, currentPosition.Margin.IsZero())       // always zero
				assert.True(t, currentPosition.OpenNotional.IsZero()) // always zero
				assert.True(t, currentPosition.LastUpdateCumulativePremiumFraction.IsZero())
				assert.EqualValues(t, 0, currentPosition.LiquidityHistoryIndex)
				assert.EqualValues(t, ctx.BlockHeight(), currentPosition.BlockNumber)
			},
		},
		{
			/*
				- long position
				- remaining margin less than liquidation fee but greater than zero
				- no bad debt
				- no funding payment

				- liquidation fee is 0.3
				- notional exchanged is 100 NUSD
				- liquidator gets 100 NUSD * 0.3 / 2 = 15 NUSD
				- position only has 10 NUSD margin, so bad debt accrues
				- ecosystem fund gets nothing (0 NUSD)
			*/
			name: "remaining margin less than liquidation fee but greater than zero",
			test: func() {
				t.Log("setup variables")
				perpKeeper, mocks, ctx := getKeeper(t)
				liquidatorAddr := sample.AccAddress()
				traderAddr := sample.AccAddress()
				pair := common.TokenPair("BTC:NUSD")

				t.Log("mock account keeper")
				vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				perpEFAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.VaultModuleAccount).
					Return(vaultAddr)
				mocks.mockAccountKeeper.EXPECT().GetModuleAddress(
					types.PerpEFModuleAccount).
					Return(perpEFAddr)

				t.Log("mock bank keeper")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.PerpEFModuleAccount, liquidatorAddr,
					sdk.NewCoins(sdk.NewInt64Coin("NUSD", 15)),
				).Return(nil)

				t.Log("setup perp keeper params")
				newParams := types.DefaultParams()
				newParams.LiquidationFee = 300_000 // liquidation fee ratio is 0.3
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
						/*baseAssetAmount=*/ sdk.NewDec(100),
					).
					Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)
				mocks.mockVpoolKeeper.EXPECT().
					SwapBaseForQuote(
						ctx,
						pair,
						/*baseAssetDirection=*/ vpooltypes.Direction_ADD_TO_POOL,
						/*baseAssetAmount=*/ sdk.NewDec(100),
						/*quoteAssetAssetLimit=*/ sdk.ZeroDec(),
					).Return( /*quoteAssetAmount=*/ sdk.NewDec(100), nil)

				t.Log("create and set the initial position")
				position := types.Position{
					Address:                             traderAddr.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(100),
					Margin:                              sdk.NewDec(10),
					OpenNotional:                        sdk.NewDec(100),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					LiquidityHistoryIndex:               0,
					BlockNumber:                         ctx.BlockHeight(),
				}
				perpKeeper.SetPosition(ctx, pair, traderAddr.String(), &position)

				t.Log("execute full liquidation")
				liquidationResp, err := perpKeeper.ExecuteFullLiquidation(ctx, liquidatorAddr, &position)
				require.NoError(t, err)

				t.Log("assert liquidation response fields")
				assert.EqualValues(t, sdk.NewDec(5), liquidationResp.BadDebt)
				assert.EqualValues(t, sdk.NewDec(15), liquidationResp.FeeToLiquidator)
				assert.EqualValues(t, sdk.ZeroDec(), liquidationResp.FeeToPerpEcosystemFund)
				assert.EqualValues(t, liquidatorAddr, liquidationResp.Liquidator)

				t.Log("assert position response fields")
				positionResp := liquidationResp.PositionResp
				assert.EqualValues(t, sdk.NewDec(100), positionResp.ExchangedQuoteAssetAmount) // amount of quote obtained
				assert.EqualValues(t, sdk.NewDec(-100), positionResp.ExchangedPositionSize)    // sold back to vpool
				assert.EqualValues(t, sdk.NewDec(-10), positionResp.MarginToVault)             // ( 10(oldMargin) + 0(unrealzedPnL) - 0(fundingPayment) ) * -1
				assert.True(t, positionResp.BadDebt.IsZero())
				assert.True(t, positionResp.FundingPayment.IsZero())
				assert.True(t, positionResp.RealizedPnl.IsZero())
				assert.True(t, positionResp.UnrealizedPnlAfter.IsZero())

				t.Log("assert current position fields")
				currentPosition := positionResp.Position
				assert.EqualValues(t, traderAddr.String(), currentPosition.Address)
				assert.EqualValues(t, pair.String(), currentPosition.Pair)
				assert.True(t, currentPosition.Size_.IsZero())        // always zero
				assert.True(t, currentPosition.Margin.IsZero())       // always zero
				assert.True(t, currentPosition.OpenNotional.IsZero()) // always zero
				assert.EqualValues(t, sdk.ZeroDec(), currentPosition.LastUpdateCumulativePremiumFraction)
				assert.EqualValues(t, 0, currentPosition.LiquidityHistoryIndex)
				assert.EqualValues(t, ctx.BlockHeight(), currentPosition.BlockNumber)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
