package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil/sample"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
				Creator: sample.AccAddress(),
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

func TestMsgMintResponse(t *testing.T) {
	tests := []struct {
		name            string
		msg             types.MsgMint
		msgResponse     types.MsgMintResponse
		collateralPrice sdk.Dec

		err error
	}{
		{
			name: "Invalid account address.",
			msg: types.MsgMint{
				Creator: "invalid_address",
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "No tokens requested",
			msg: types.MsgMint{
				Creator: sample.AccAddress(),
				Stable:  sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			msgResponse: types.MsgMintResponse{
				Stable: sdk.NewCoin("usdm", sdk.NewInt(0)),
			},
			err: nil,
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
