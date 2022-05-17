package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRemoveMargin{}
var _ sdk.Msg = &MsgAddMargin{}
var _ sdk.Msg = &MsgLiquidate{}

// MsgRemoveMargin

func (m MsgRemoveMargin) Route() string { return RouterKey }
func (m MsgRemoveMargin) Type() string  { return "remove_margin_msg" }

func (m MsgRemoveMargin) ValidateBasic() error {
	return nil
}

func (m MsgRemoveMargin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgRemoveMargin) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// MsgAddMargin

func (m MsgAddMargin) Route() string { return RouterKey }
func (m MsgAddMargin) Type() string  { return "add_margin_msg" }

func (m MsgAddMargin) ValidateBasic() error {
	return nil
}

func (m MsgAddMargin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgAddMargin) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// MsgLiquidate

func (m MsgLiquidate) Route() string { return RouterKey }
func (m MsgLiquidate) Type() string  { return "add_margin_msg" }

func (m MsgLiquidate) ValidateBasic() error {
	return nil
}

func (m MsgLiquidate) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgLiquidate) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}
