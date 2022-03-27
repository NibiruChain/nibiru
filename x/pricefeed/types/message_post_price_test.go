package types

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
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
				From: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgPostPrice{
				From: sample.AccAddress(),
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
