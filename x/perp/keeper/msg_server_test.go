package keeper_test

import (
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

			t.Log("fund trader")
			simapp.FundAccount(app.BankKeeper, ctx, sdk.MustAccAddressFromBech32(tc.sender), tc.traderFunds)

			resp, err := msgServer.OpenPosition(sdk.WrapSDKContext(ctx), &types.MsgOpenPosition{
				Sender:               tc.sender,
				TokenPair:            tc.pair,
				Side:                 types.Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(1000),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.ZeroInt(),
			})

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
