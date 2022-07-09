package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
			name:        "invalid pair",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			pair:        "foo",
			sender:      sample.AccAddress().String(),
			expectedErr: common.ErrInvalidTokenPair,
		},
		{
			name:        "invalid address",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			pair:        common.PairBTCStable.String(),
			sender:      "bar",
			expectedErr: fmt.Errorf("decoding bech32 failed"),
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
				simapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds)
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
			name:        "invalid pair",
			pair:        "foo",
			sender:      sample.AccAddress().String(),
			expectedErr: common.ErrInvalidTokenPair,
		},
		{
			name:        "invalid address",
			pair:        common.PairBTCStable.String(),
			sender:      "foo",
			expectedErr: fmt.Errorf("decoding bech32 failed"),
		},
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
				app.PerpKeeper.PositionsState(ctx).Create(&types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                pair,
					Size_:                               sdk.OneDec(),
					Margin:                              sdk.OneDec(),
					OpenNotional:                        sdk.OneDec(),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         1,
				})
				simapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(pair.GetQuoteTokenDenom(), 1)))
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
