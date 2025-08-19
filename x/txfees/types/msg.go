package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

var (
	_ legacytx.LegacyMsg = &MsgUpdateFeeToken{}
	_ legacytx.LegacyMsg = &MsgUpdateParams{}
)

const TypeMsgUpdateFeeToken = "update_fee_token"

// Route implements legacytx.LegacyMsg
func (msg MsgUpdateFeeToken) Route() string { return RouterKey }

func (msg MsgUpdateFeeToken) Type() string { return TypeMsgUpdateFeeToken }

func (msg MsgUpdateFeeToken) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateFeeToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateFeeToken) ValidateBasic() error {
	// TODO
	return nil
}

const TypeMsgUpdateParams = "update_params"

// Route implements legacytx.LegacyMsg
func (msg MsgUpdateParams) Route() string { return RouterKey }

func (msg MsgUpdateParams) Type() string { return TypeMsgUpdateParams }

func (msg MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateParams) ValidateBasic() error {
	// TODO
	return nil
}
