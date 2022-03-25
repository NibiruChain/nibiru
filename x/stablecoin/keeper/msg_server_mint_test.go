package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/testutil"
	"github.com/MatrixDao/matrix/x/stablecoin/types"

	// sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMint_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  types.MsgMint
		err  error
	}{
		{
			name: "invalid address",
			msg: types.MsgMint{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: types.MsgMint{
				Creator: testutil.SampleAccAddress(),
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
