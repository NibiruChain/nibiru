package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

// ----------------------------------------------------------------
// MsgCreateDenom

var _ sdk.Msg = &MsgCreateDenom{}
var _ legacytx.LegacyMsg = &MsgCreateDenom{}

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgCreateDenom) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return ErrInvalidCreator.Wrapf("%s: sender address (%s)", err.Error(), m.Sender)
	}

	denom := TFDenom{
		Creator:  m.Sender,
		Subdenom: m.Subdenom,
	}
	err = denom.Validate()
	if err != nil {
		return ErrInvalidDenom.Wrap(err.Error())
	}

	return nil
}

// GetSigners: Impl sdk.Msg.
func (m MsgCreateDenom) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgCreateDenom) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgCreateDenom) Type() string { return "create_denom" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgCreateDenom) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgChangeAdmin

var _ sdk.Msg = &MsgChangeAdmin{}
var _ legacytx.LegacyMsg = &MsgChangeAdmin{}

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgChangeAdmin) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid sender (%s): %s", m.Sender, err)
	}

	_, err = sdk.AccAddressFromBech32(m.NewAdmin)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid new admin (%s): %s", m.NewAdmin, err)
	}

	return DenomStr(m.Denom).Validate()
}

// GetSigners: Impl sdk.Msg.
func (m MsgChangeAdmin) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgChangeAdmin) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgChangeAdmin) Type() string { return "create_denom" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgChangeAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgUpdateModuleParams

var _ sdk.Msg = &MsgUpdateModuleParams{}

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgUpdateModuleParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid authority (%s): %s", m.Authority, err)
	}
	return m.Params.Validate()
}

// GetSigners: Impl sdk.Msg.
func (m MsgUpdateModuleParams) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{sender}
}
