package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	nibisimapp "github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestMsgServerAddMargin(t *testing.T) {
	tests := []struct {
		name string

		traderFunds     sdk.Coins
		initialPosition *types.Position
		margin          sdk.Coin

		expectedErr error
	}{
		{
			name:        "trader not enough funds",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.OneDec(),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			margin:      sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr: sdkerrors.ErrInsufficientFunds,
		},
		{
			name:            "no initial position",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
			initialPosition: nil,
			margin:          sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr:     collections.ErrNotFound,
		},
		{
			name:        "success",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.OneDec(),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			margin:      sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)
			traderAddr := testutil.AccAddress()

			t.Log("create vpool")
			assert.NoError(t, app.VpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteReserve */ sdk.NewDec(1*common.Precision),
				/* baseReserve */ sdk.NewDec(1*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("fund trader")
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("create position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				setPosition(app.PerpKeeper, ctx, *tc.initialPosition)
			}

			resp, err := msgServer.AddMargin(sdk.WrapSDKContext(ctx), &types.MsgAddMargin{
				Sender: traderAddr.String(),
				Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Margin: tc.margin,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.EqualValues(t, resp.FundingPayment, sdk.ZeroDec())
				assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, tc.initialPosition.Margin.Add(tc.margin.Amount.ToDec()), resp.Position.Margin)
				assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
				assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
			}
		})
	}
}

func TestMsgServerRemoveMargin(t *testing.T) {
	tests := []struct {
		name string

		vaultFunds      sdk.Coins
		initialPosition *types.Position
		marginToRemove  sdk.Coin

		expectedErr error
	}{
		{
			name:       "position not enough margin",
			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.OneDec(),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr:    types.ErrFailedRemoveMarginCanCauseBadDebt,
		},
		{
			name:            "no initial position",
			vaultFunds:      sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 0)),
			initialPosition: nil,
			marginToRemove:  sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr:     collections.ErrNotFound,
		},
		{
			name:       "vault insufficient funds",
			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1 * common.Precision),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr:    sdkerrors.ErrInsufficientFunds,
		},
		{
			name:       "success",
			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1 * common.Precision),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
			expectedErr:    nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)
			traderAddr := testutil.AccAddress()

			t.Log("create vpool")
			assert.NoError(t, app.VpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteReserve */ sdk.NewDec(1*common.Precision),
				/* baseReserve */ sdk.NewDec(1*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("fund vault")
			require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, tc.vaultFunds))

			if tc.initialPosition != nil {
				t.Log("create position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				setPosition(app.PerpKeeper, ctx, *tc.initialPosition)
			}

			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 5)).WithBlockHeight(ctx.BlockHeight() + 1)

			resp, err := msgServer.RemoveMargin(sdk.WrapSDKContext(ctx), &types.MsgRemoveMargin{
				Sender: traderAddr.String(),
				Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Margin: tc.marginToRemove,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.EqualValues(t, tc.marginToRemove, resp.MarginOut)
				assert.EqualValues(t, resp.FundingPayment, sdk.ZeroDec())
				assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
				assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
				assert.EqualValues(t, tc.initialPosition.Margin.Sub(tc.marginToRemove.Amount.ToDec()), resp.Position.Margin)
				assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
				assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
			}
		})
	}
}

func TestMsgServerOpenPosition(t *testing.T) {
	tests := []struct {
		name        string
		traderFunds sdk.Coins
		pair        common.AssetPair
		sender      string
		expectedErr error
	}{
		{
			name:        "trader not enough funds",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			sender:      testutil.AccAddress().String(),
			expectedErr: sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "success",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			sender:      testutil.AccAddress().String(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")
			assert.NoError(t, app.VpoolKeeper.CreatePool(
				/* ctx */ ctx,
				/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteAssetReserve */ sdk.NewDec(1*common.Precision),
				/* baseAssetReserve */ sdk.NewDec(1*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			traderAddr, err := sdk.AccAddressFromBech32(tc.sender)
			if err == nil {
				t.Log("fund trader")
				require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))
			}

			t.Log("increment block height and time for TWAP calculation")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
				WithBlockTime(time.Now().Add(time.Minute))

			resp, err := msgServer.OpenPosition(sdk.WrapSDKContext(ctx), &types.MsgOpenPosition{
				Sender:               tc.sender,
				Pair:                 tc.pair,
				Side:                 types.Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(1000),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.ZeroInt(),
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.EqualValues(t, tc.pair, resp.Position.Pair.String())
				assert.EqualValues(t, tc.sender, resp.Position.TraderAddress)
				assert.EqualValues(t, sdk.MustNewDecFromStr("9900.990099009900990099"), resp.Position.Size_)
				assert.EqualValues(t, sdk.NewDec(1000), resp.Position.Margin)
				assert.EqualValues(t, sdk.NewDec(10_000), resp.Position.OpenNotional)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
				assert.EqualValues(t, sdk.NewDec(10_000), resp.ExchangedNotionalValue)
				assert.EqualValues(t, sdk.MustNewDecFromStr("9900.990099009900990099"), resp.ExchangedPositionSize)
				assert.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
				assert.EqualValues(t, sdk.ZeroDec(), resp.RealizedPnl)
				assert.EqualValues(t, sdk.ZeroDec(), resp.UnrealizedPnlAfter)
				assert.EqualValues(t, sdk.NewDec(1000), resp.MarginToVault)
				assert.EqualValues(t, sdk.NewDec(10_000), resp.PositionNotional)
			}
		})
	}
}

func TestMsgServerClosePosition(t *testing.T) {
	tests := []struct {
		name string

		pair       common.AssetPair
		traderAddr sdk.AccAddress

		expectedErr error
	}{
		{
			name:        "success",
			pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			traderAddr:  testutil.AccAddress(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")

			assert.NoError(t, app.VpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteAssetReserve */ sdk.NewDec(1*common.Precision),
				/* baseAssetReserve */ sdk.NewDec(1*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("create position")
			setPosition(app.PerpKeeper, ctx, types.Position{
				TraderAddress:                   tc.traderAddr.String(),
				Pair:                            tc.pair,
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.OneDec(),
				OpenNotional:                    sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			})
			require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(tc.pair.QuoteDenom(), 1))))

			resp, err := msgServer.ClosePosition(sdk.WrapSDKContext(ctx), &types.MsgClosePosition{
				Sender: tc.traderAddr.String(),
				Pair:   tc.pair,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.999999000000999999"), resp.ExchangedNotionalValue)
				assert.EqualValues(t, sdk.NewDec(-1), resp.ExchangedPositionSize)
				assert.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.999999000000999999"), resp.MarginToTrader)
				assert.EqualValues(t, sdk.MustNewDecFromStr("-0.000000999999000001"), resp.RealizedPnl)
			}
		})
	}
}

func TestMsgServerLiquidate(t *testing.T) {
	tests := []struct {
		name string

		pair       common.AssetPair
		liquidator string
		trader     string

		expectedErr error
	}{
		{
			name:        "success",
			pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			liquidator:  testutil.AccAddress().String(),
			trader:      testutil.AccAddress().String(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())
			setLiquidator(ctx, app.PerpKeeper, tc.liquidator)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")
			assert.NoError(t, app.VpoolKeeper.CreatePool(
				/* ctx */ ctx,
				/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteAssetReserve */ sdk.NewDec(1*common.Precision),
				/* baseAssetReserve */ sdk.NewDec(1*common.Precision),
				vpooltypes.VpoolConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
			))
			setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

			traderAddr, err := sdk.AccAddressFromBech32(tc.trader)
			if err == nil {
				t.Log("set oracle price")
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.OneDec())

				t.Log("create position")
				setPosition(app.PerpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            tc.pair,
					Size_:                           sdk.OneDec(),
					Margin:                          sdk.OneDec(),
					OpenNotional:                    sdk.NewDec(2), // new spot price is 1, so position can be liquidated
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     1,
				})
				require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(tc.pair.QuoteDenom(), 1))))
			}

			resp, err := msgServer.Liquidate(sdk.WrapSDKContext(ctx), &types.MsgLiquidate{
				Sender: tc.liquidator,
				Pair:   tc.pair,
				Trader: tc.trader,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}

func setLiquidator(ctx sdk.Context, perpKeeper keeper.Keeper, liquidator string) {
	p := perpKeeper.GetParams(ctx)
	p.WhitelistedLiquidators = []string{liquidator}
	perpKeeper.SetParams(ctx, p)
}

func TestMsgServerMultiLiquidate(t *testing.T) {
	app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
	ctx = ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	liquidator := testutil.AccAddress()

	atRiskTrader1 := testutil.AccAddress()
	notAtRiskTrader := testutil.AccAddress()
	atRiskTrader2 := testutil.AccAddress()

	t.Log("create vpool")
	assert.NoError(t, app.VpoolKeeper.CreatePool(
		/* ctx */ ctx,
		/* pair */ pair,
		/* quoteAssetReserve */ sdk.NewDec(1*common.Precision),
		/* baseAssetReserve */ sdk.NewDec(1*common.Precision),
		vpooltypes.VpoolConfig{
			TradeLimitRatio:        sdk.OneDec(),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	))
	setPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
		Pair:                            pair,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	})
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

	t.Log("set oracle price")
	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.OneDec())

	t.Log("create positions")
	atRiskPosition1 := types.Position{
		TraderAddress:                   atRiskTrader1.String(),
		Pair:                            pair,
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.NewDec(2),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
	}
	atRiskPosition2 := types.Position{
		TraderAddress:                   atRiskTrader2.String(),
		Pair:                            pair,
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.NewDec(2), // new spot price is 1, so position can be liquidated
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		BlockNumber:                     1,
	}
	notAtRiskPosition := types.Position{
		TraderAddress:                   notAtRiskTrader.String(),
		Pair:                            pair,
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.MustNewDecFromStr("0.1"), // open price is lower than current price so no way trader gets liquidated
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		BlockNumber:                     1,
	}
	setPosition(app.PerpKeeper, ctx, atRiskPosition1)
	setPosition(app.PerpKeeper, ctx, notAtRiskPosition)
	setPosition(app.PerpKeeper, ctx, atRiskPosition2)

	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 2))))

	setLiquidator(ctx, app.PerpKeeper, liquidator.String())
	resp, err := msgServer.MultiLiquidate(sdk.WrapSDKContext(ctx), &types.MsgMultiLiquidate{
		Sender: liquidator.String(),
		Liquidations: []*types.MsgMultiLiquidate_MultiLiquidation{
			{
				Pair:   pair,
				Trader: atRiskTrader1.String(),
			},
			{
				Pair:   pair,
				Trader: notAtRiskTrader.String(),
			},
			{
				Pair:   pair,
				Trader: atRiskTrader2.String(),
			},
		},
	})
	require.NoError(t, err)

	_, successLiq := resp.LiquidationResponses[0].Response.(*types.MsgMultiLiquidateResponse_MultiLiquidateResponse_Liquidation)
	require.True(t, successLiq)

	_, unsuccessfulLiq := resp.LiquidationResponses[1].Response.(*types.MsgMultiLiquidateResponse_MultiLiquidateResponse_Error)
	require.True(t, unsuccessfulLiq)

	_, successLiq = resp.LiquidationResponses[2].Response.(*types.MsgMultiLiquidateResponse_MultiLiquidateResponse_Liquidation)
	require.True(t, successLiq)

	// NOTE: we don't care about checking if liquidations math is correct. This is the duty of keeper.Liquidate
	// what we care about is that the first and third liquidations made some modifications at state
	// and events levels, whilst the second (which failed) didn't.

	assertNotLiquidated := func(old types.Position) {
		position, err := app.PerpKeeper.Positions.Get(ctx, collections.Join(old.Pair, sdk.MustAccAddressFromBech32(old.TraderAddress)))
		require.NoError(t, err)
		require.Equal(t, old, position)
	}

	assertLiquidated := func(old types.Position) {
		_, err := app.PerpKeeper.Positions.Get(ctx, collections.Join(old.Pair, sdk.MustAccAddressFromBech32(old.TraderAddress)))
		require.ErrorIs(t, err, collections.ErrNotFound)
		// NOTE(mercilex): does not cover partial liquidation
	}
	assertNotLiquidated(notAtRiskPosition)
	assertLiquidated(atRiskPosition1)
	assertLiquidated(atRiskPosition2)
}

func TestMsgServerDonateToEcosystemFund(t *testing.T) {
	tests := []struct {
		name string

		sender       sdk.AccAddress
		initialFunds sdk.Coins
		donation     sdk.Coin

		expectedErr error
	}{
		{
			name:         "not enough funds",
			sender:       testutil.AccAddress(),
			initialFunds: sdk.NewCoins(),
			donation:     sdk.NewInt64Coin(denoms.NUSD, 100),
			expectedErr:  fmt.Errorf("insufficient funds"),
		},
		{
			name:         "success",
			sender:       testutil.AccAddress(),
			initialFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e6)),
			donation:     sdk.NewInt64Coin(denoms.NUSD, 100),
			expectedErr:  nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)
			require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, tc.sender, tc.initialFunds))

			resp, err := msgServer.DonateToEcosystemFund(sdk.WrapSDKContext(ctx), &types.MsgDonateToEcosystemFund{
				Sender:   tc.sender.String(),
				Donation: tc.donation,
			})

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.EqualValues(t,
					tc.donation,
					app.BankKeeper.GetBalance(
						ctx,
						app.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount),
						denoms.NUSD,
					),
				)
			}
		})
	}
}
