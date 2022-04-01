package types

import (
	"testing"

	"github.com/MatrixDao/matrix/x/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

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
				Creator: sample.AccAddress().String(),
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
