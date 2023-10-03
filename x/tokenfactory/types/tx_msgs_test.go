package types_test

import (
	fmt "fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func validateBasicTest(msg sdk.Msg, wantErr string) func(t *testing.T) {
	return func(t *testing.T) {
		err := msg.ValidateBasic()
		if wantErr != "" {
			assert.Error(t, err)
			require.ErrorContains(t, err, wantErr)
		} else {
			require.NoError(t, err)
		}
	}
}

type ValidateBasicTest struct {
	name    string
	msg     sdk.Msg
	wantErr string
}

func TestMsgCreateDenom_ValidateBasic(t *testing.T) {
	addr := testutil.AccAddress().String()
	for _, tc := range []ValidateBasicTest{
		{
			name: "happy",
			msg: &types.MsgCreateDenom{
				Sender:   addr,
				Subdenom: "subdenom",
			},
			wantErr: "",
		},
		{
			name: "sad subdenom",
			msg: &types.MsgCreateDenom{
				Sender:   addr,
				Subdenom: "",
			},
			wantErr: "empty subdenom",
		},
		{
			name: "sad creator",
			msg: &types.MsgCreateDenom{
				Sender:   "creator",
				Subdenom: "subdenom",
			},
			wantErr: "invalid creator",
		},
	} {
		t.Run(tc.name, validateBasicTest(tc.msg, tc.wantErr))
	}
}

func TestMsgChangeAdmin_ValidateBasic(t *testing.T) {
	sbf := testutil.AccAddress().String()
	validDenom := fmt.Sprintf("tf/%s/ftt", sbf)
	for _, tc := range []ValidateBasicTest{
		{
			name: "happy",
			msg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    validDenom,
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: "",
		},
		{
			name: "invalid sender",
			msg: &types.MsgChangeAdmin{
				Sender:   "sender",
				Denom:    validDenom,
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: "invalid sender",
		},
		{
			name: "invalid new admin",
			msg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    validDenom,
				NewAdmin: "new-admin",
			},
			wantErr: "invalid new admin",
		},
		{
			name: "invalid denom",
			msg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    "tf/",
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: "denom format error",
		},
	} {
		t.Run(tc.name, validateBasicTest(tc.msg, tc.wantErr))
	}
}

func TestMsgUpdateModuleParams_ValidateBasic(t *testing.T) {
	for _, tc := range []ValidateBasicTest{
		{
			name: "happy",
			msg: &types.MsgUpdateModuleParams{
				Authority: testutil.AccAddress().String(),
				Params:    types.DefaultModuleParams(),
			},
			wantErr: "",
		},
		{
			name: "sad authority",
			msg: &types.MsgUpdateModuleParams{
				Authority: "authority",
				Params:    types.DefaultModuleParams(),
			},
			wantErr: "invalid authority",
		},
	} {
		t.Run(tc.name, validateBasicTest(tc.msg, tc.wantErr))
	}
}

func TestTxMsgInterface(t *testing.T) {
	creator := testutil.AccAddress().String()
	subdenom := testutil.RandLetters(4)
	for _, msg := range []legacytx.LegacyMsg{
		&types.MsgCreateDenom{
			Sender:   creator,
			Subdenom: subdenom,
		},
		&types.MsgChangeAdmin{
			Sender:   creator,
			Denom:    fmt.Sprintf("tf/%s/%s", creator, subdenom),
			NewAdmin: testutil.AccAddress().String(),
		},
	} {
		t.Run(msg.Type(), func(t *testing.T) {
			require.NotPanics(t, func() {
				_ = msg.GetSigners()
				_ = msg.Route()
				_ = msg.Type()
				_ = msg.GetSignBytes()
			})
		})
	}

	for _, msg := range []sdk.Msg{
		&types.MsgUpdateModuleParams{
			Authority: testutil.GovModuleAddr().String(),
			Params:    types.DefaultModuleParams(),
		},
	} {
		require.NotPanics(t, func() {
			_ = msg.GetSigners()
		})
	}
}
