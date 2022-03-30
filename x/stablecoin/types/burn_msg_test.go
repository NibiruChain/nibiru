package types

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgBurn_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgBurnStable
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgBurnStable{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgBurnStable{
				Creator: sample.AccAddress().String(),
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
