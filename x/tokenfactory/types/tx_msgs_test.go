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
	denomStr := fmt.Sprintf("tf/%s/%s", creator, subdenom)
	for _, msg := range []legacytx.LegacyMsg{
		&types.MsgCreateDenom{
			Sender:   creator,
			Subdenom: subdenom,
		},
		&types.MsgChangeAdmin{
			Sender:   creator,
			Denom:    denomStr,
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
		&types.MsgMint{
			Sender: creator,
			Coin:   sdk.NewInt64Coin(denomStr, 420),
			MintTo: "",
		},
		&types.MsgBurn{
			Sender:   creator,
			Coin:     sdk.NewInt64Coin(denomStr, 420),
			BurnFrom: "",
		},
	} {
		require.NotPanics(t, func() {
			_ = msg.GetSigners()
		})
	}
}

func TestMsgMint_ValidateBasic(t *testing.T) {
	sbf := testutil.AccAddress().String()
	validDenom := fmt.Sprintf("tf/%s/ftt", sbf)
	validCoin := sdk.NewInt64Coin(validDenom, 420)
	for _, tc := range []ValidateBasicTest{
		{
			name: "happy",
			msg: &types.MsgMint{
				Sender: sbf,
				Coin:   validCoin,
				MintTo: "",
			},
			wantErr: "",
		},
		{
			name: "invalid sender",
			msg: &types.MsgMint{
				Sender: "sender",
				Coin:   validCoin,
				MintTo: "",
			},
			wantErr: "invalid address",
		},
		{
			name: "invalid denom",
			msg: &types.MsgMint{
				Sender: sbf,
				Coin:   sdk.Coin{Denom: "tf/", Amount: sdk.NewInt(420)},
				MintTo: "",
			},
			wantErr: "denom format error",
		},
		{
			name: "invalid mint to addr",
			msg: &types.MsgMint{
				Sender: sbf,
				Coin:   validCoin,
				MintTo: "mintto",
			},
			wantErr: "invalid mint_to",
		},
		{
			name: "invalid coin",
			msg: &types.MsgMint{
				Sender: sbf,
				Coin:   sdk.Coin{Amount: sdk.NewInt(-420)},
				MintTo: "",
			},
			wantErr: "invalid coin",
		},
	} {
		t.Run(tc.name, validateBasicTest(tc.msg, tc.wantErr))
	}
}

func TestMsgBurn_ValidateBasic(t *testing.T) {
	sbf := testutil.AccAddress().String()
	validDenom := fmt.Sprintf("tf/%s/ftt", sbf)
	validCoin := sdk.NewInt64Coin(validDenom, 420)
	for _, tc := range []ValidateBasicTest{
		{
			name: "happy",
			msg: &types.MsgBurn{
				Sender:   sbf,
				Coin:     validCoin,
				BurnFrom: "",
			},
			wantErr: "",
		},
		{
			name: "invalid sender",
			msg: &types.MsgBurn{
				Sender:   "sender",
				Coin:     validCoin,
				BurnFrom: "",
			},
			wantErr: "invalid address",
		},
		{
			name: "invalid denom",
			msg: &types.MsgBurn{
				Sender:   sbf,
				Coin:     sdk.Coin{Denom: "tf/", Amount: sdk.NewInt(420)},
				BurnFrom: "",
			},
			wantErr: "denom format error",
		},
		{
			name: "invalid burn from addr",
			msg: &types.MsgBurn{
				Sender:   sbf,
				Coin:     validCoin,
				BurnFrom: "mintto",
			},
			wantErr: "invalid burn_from",
		},
		{
			name: "invalid coin",
			msg: &types.MsgBurn{
				Sender:   sbf,
				Coin:     sdk.Coin{Amount: sdk.NewInt(-420)},
				BurnFrom: "",
			},
			wantErr: "invalid coin",
		},
	} {
		t.Run(tc.name, validateBasicTest(tc.msg, tc.wantErr))
	}
}
