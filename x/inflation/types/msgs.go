package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ensure Msg interface compliance at compile time
var (
	_ sdk.Msg = &MsgEditInflationParams{}
	_ sdk.Msg = &MsgToggleInflation{}
)

// oracle message types
const (
	TypeMsgEditInflationParams = "edit_inflation_params"
	TypeMsgToggleInflation     = "toggle_inflation"
)

// Route implements sdk.Msg
func (msg MsgEditInflationParams) Route() string { return RouterKey }

// Type implements sdk.Msg
func (msg MsgEditInflationParams) Type() string { return TypeMsgEditInflationParams }

// GetSignBytes implements sdk.Msg
func (msg MsgEditInflationParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgEditInflationParams) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

func (m MsgEditInflationParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}
	return nil
}

// -------------------------------------------------
// MsgToggleInflation
// Route implements sdk.Msg
func (msg MsgToggleInflation) Route() string { return RouterKey }

// Type implements sdk.Msg
func (msg MsgToggleInflation) Type() string { return TypeMsgToggleInflation }

// GetSignBytes implements sdk.Msg
func (msg MsgToggleInflation) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgToggleInflation) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

func (m MsgToggleInflation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}
	return nil
}
