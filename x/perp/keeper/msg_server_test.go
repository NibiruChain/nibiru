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

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestMsgServerOpenPosition(t *testing.T) {
	tests := []struct {
		name string

		traderFunds sdk.Coins
		pair        string
		sender      string

		expectedErr error
	}{
		{
			name:        "trader not enough funds",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 999)),
			pair:        common.PairBTCStable.String(),
			sender:      sample.AccAddress().String(),
			expectedErr: sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "success",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			pair:        common.PairBTCStable.String(),
			sender:      sample.AccAddress().String(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")
			app.VpoolKeeper.CreatePool(ctx, common.PairBTCStable, sdk.OneDec(), sdk.NewDec(1_000_000), sdk.NewDec(1_000_000), sdk.OneDec(), sdk.OneDec())
			app.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})

			traderAddr, err := sdk.AccAddressFromBech32(tc.sender)
			if err == nil {
				t.Log("fund trader")
				require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))
			}

			resp, err := msgServer.OpenPosition(sdk.WrapSDKContext(ctx), &types.MsgOpenPosition{
				Sender:               tc.sender,
				TokenPair:            tc.pair,
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
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LastUpdateCumulativePremiumFraction)
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

		pair   string
		sender string

		expectedErr error
	}{
		{
			name:        "success",
			pair:        common.PairBTCStable.String(),
			sender:      sample.AccAddress().String(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")
			app.VpoolKeeper.CreatePool(ctx, common.PairBTCStable, sdk.OneDec(), sdk.NewDec(1_000_000), sdk.NewDec(1_000_000), sdk.OneDec(), sdk.OneDec())
			app.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})

			pair, err := common.NewAssetPair(tc.pair)
			traderAddr, err2 := sdk.AccAddressFromBech32(tc.sender)
			if err == nil && err2 == nil {
				t.Log("create position")
				require.NoError(t, app.PerpKeeper.PositionsState(ctx).Create(&types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                pair,
					Size_:                               sdk.OneDec(),
					Margin:                              sdk.OneDec(),
					OpenNotional:                        sdk.OneDec(),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         1,
				}))
				require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(pair.GetQuoteTokenDenom(), 1))))
			}

			resp, err := msgServer.ClosePosition(sdk.WrapSDKContext(ctx), &types.MsgClosePosition{
				Sender:    tc.sender,
				TokenPair: tc.pair,
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

func TestMsgServerLiquidate(t *testing.T) {
	tests := []struct {
		name string

		pair       string
		liquidator string
		trader     string

		expectedErr error
	}{
		{
			name:       "invalid pair",
			pair:       "foo",
			liquidator: sample.AccAddress().String(),
			trader:     sample.AccAddress().String(),

			expectedErr: common.ErrInvalidTokenPair,
		},
		{
			name:        "invalid liquidator address",
			pair:        common.PairBTCStable.String(),
			liquidator:  "foo",
			trader:      sample.AccAddress().String(),
			expectedErr: fmt.Errorf("decoding bech32 failed"),
		},
		{
			name:        "invalid trader address",
			pair:        common.PairBTCStable.String(),
			liquidator:  sample.AccAddress().String(),
			trader:      "foo",
			expectedErr: fmt.Errorf("decoding bech32 failed"),
		},
		{
			name:        "success",
			pair:        common.PairBTCStable.String(),
			liquidator:  sample.AccAddress().String(),
			trader:      sample.AccAddress().String(),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruAppAndContext(true)
			msgServer := keeper.NewMsgServerImpl(app.PerpKeeper)

			t.Log("create vpool")
			app.VpoolKeeper.CreatePool(ctx, common.PairBTCStable, sdk.OneDec(), sdk.NewDec(1_000_000), sdk.NewDec(1_000_000), sdk.OneDec(), sdk.OneDec())
			app.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

			pair, err := common.NewAssetPair(tc.pair)
			traderAddr, err2 := sdk.AccAddressFromBech32(tc.trader)
			if err == nil && err2 == nil {
				t.Log("set pricefeed oracle price")
				oracle := sample.AccAddress()
				app.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
				_, err = app.PricefeedKeeper.PostRawPrice(ctx, oracle, pair.String(), sdk.OneDec(), time.Now().Add(time.Hour))
				require.NoError(t, err)
				require.NoError(t, app.PricefeedKeeper.GatherRawPrices(ctx, pair.GetBaseTokenDenom(), pair.GetQuoteTokenDenom()))

				t.Log("create position")
				require.NoError(t, app.PerpKeeper.PositionsState(ctx).Create(&types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                pair,
					Size_:                               sdk.OneDec(),
					Margin:                              sdk.OneDec(),
					OpenNotional:                        sdk.NewDec(2), // new spot price is 1, so position can be liquidated
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         1,
				}))
				require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(pair.GetQuoteTokenDenom(), 1))))
			}

			resp, err := msgServer.Liquidate(sdk.WrapSDKContext(ctx), &types.MsgLiquidate{
				Sender:    tc.liquidator,
				TokenPair: tc.pair,
				Trader:    tc.trader,
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
