package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestMsgPostPrice_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgPostPrice
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgPostPrice{
				Oracle: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgPostPrice{
				Oracle: sample.AccAddress().String(),
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
