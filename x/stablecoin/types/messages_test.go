package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMintStable_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgMintStable
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgMintStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgMintStable{
				Creator: testutil.AccAddress().String(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgBurn_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msgBurn MsgBurnStable
		err     error
	}{
		{
			name: "Invalid MsgBurn.Creator address",
			msgBurn: MsgBurnStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "Valid MsgBurn.Creator address",
			msgBurn: MsgBurnStable{
				Creator: testutil.AccAddress().String(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.msgBurn.ValidateBasic()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgRecollateralize_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRecollateralize
		err  error
	}{
		{
			name: "Invalid MsgBurn.Creator address",
			msg: MsgRecollateralize{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "Valid MsgBurn.Creator address",
			msg: MsgRecollateralize{
				Creator: testutil.AccAddress().String(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
