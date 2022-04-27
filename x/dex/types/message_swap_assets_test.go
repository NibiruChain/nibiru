package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgSwapAssets_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSwapAssets
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSwapAssets{
				Sender:        "invalid_address",
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid pool id",
			msg: MsgSwapAssets{
				Sender:        sample.AccAddress().String(),
				PoolId:        0,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
			err: ErrInvalidPoolId,
		},
		{
			name: "invalid tokens in",
			msg: MsgSwapAssets{
				Sender:        sample.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 0),
				TokenOutDenom: "bar",
			},
			err: ErrInvalidTokenIn,
		},
		{
			name: "invalid token out denom",
			msg: MsgSwapAssets{
				Sender:        sample.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "",
			},
			err: ErrInvalidTokenOutDenom,
		},
		{
			name: "valid message",
			msg: MsgSwapAssets{
				Sender:        sample.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
