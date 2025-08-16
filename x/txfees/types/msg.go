package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

var (
	_ legacytx.LegacyMsg = &MsgUpdateFeeToken{}
)

const TypeMsgUpdateFeeToken = "update_fee_token"

// Route implements legacytx.LegacyMsg
func (msg MsgUpdateFeeToken) Route() string { return RouterKey }

func (msg MsgUpdateFeeToken) Type() string { return TypeMsgUpdateFeeToken }

func (msg MsgUpdateFeeToken) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateFeeToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateFeeToken) ValidateBasic() error {
	// TODO
	return nil
}
