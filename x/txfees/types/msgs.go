package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

var (
	_ legacytx.LegacyMsg = &MsgSetFeeTokens{}
)

const TypeMsgSetFeeTokens = "set_fee_tokens"

// Route implements legacytx.LegacyMsg
func (msg MsgSetFeeTokens) Route() string { return RouterKey }

func (msg MsgSetFeeTokens) Type() string { return TypeMsgSetFeeTokens }

func (msg MsgSetFeeTokens) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgSetFeeTokens) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

func (m MsgSetFeeTokens) ValidateBasic() error {
	// TODO
	return nil
}
