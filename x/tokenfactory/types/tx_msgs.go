package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

// ----------------------------------------------------------------
// MsgCreateDenom

var (
	_ legacytx.LegacyMsg = &MsgCreateDenom{}
	_ legacytx.LegacyMsg = &MsgChangeAdmin{}
	_ legacytx.LegacyMsg = &MsgUpdateModuleParams{}
	_ legacytx.LegacyMsg = &MsgMint{}
	_ legacytx.LegacyMsg = &MsgBurn{}
	_ legacytx.LegacyMsg = &MsgSetDenomMetadata{}
	_ legacytx.LegacyMsg = &MsgBurnNative{}
)

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
// MsgMint

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgMint) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid sender (%s): %s", m.Sender, err)
	}

	if err := validateCoin(m.Coin); err != nil {
		return err
	} else if err := DenomStr(m.Coin.Denom).Validate(); err != nil {
		return err
	}

	if m.MintTo != "" {
		_, err = sdk.AccAddressFromBech32(m.MintTo)
		if err != nil {
			return sdkerrors.ErrInvalidAddress.Wrapf(
				"invalid mint_to (%s): %s", m.MintTo, err)
		}
	}

	return err
}

// GetSigners: Impl sdk.Msg.
func (m MsgMint) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

func validateCoin(coin sdk.Coin) error {
	if !coin.IsValid() || coin.IsZero() {
		return sdkerrors.ErrInvalidCoins.Wrap(coin.String())
	}
	return nil
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgMint) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgMint) Type() string { return "mint" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgMint) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgBurn

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgBurn) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid sender (%s): %s", m.Sender, err)
	}

	if err := validateCoin(m.Coin); err != nil {
		return err
	} else if err := DenomStr(m.Coin.Denom).Validate(); err != nil {
		return err
	}

	if m.BurnFrom != "" {
		_, err = sdk.AccAddressFromBech32(m.BurnFrom)
		if err != nil {
			return sdkerrors.ErrInvalidAddress.Wrapf(
				"invalid burn_from (%s): %s", m.BurnFrom, err)
		}
	}

	return nil
}

// GetSigners: Impl sdk.Msg.
func (m MsgBurn) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgBurn) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgBurn) Type() string { return "burn" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgBurn) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgUpdateModuleParams

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

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgUpdateModuleParams) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgUpdateModuleParams) Type() string { return "update_module_params" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgUpdateModuleParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgSetDenomMetadata

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgSetDenomMetadata) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid sender (%s): %s", m.Sender, err)
	}
	return m.Metadata.Validate()
}

// GetSigners: Impl sdk.Msg.
func (m MsgSetDenomMetadata) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgSetDenomMetadata) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgSetDenomMetadata) Type() string { return "set_denom_metadata" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgSetDenomMetadata) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ----------------------------------------------------------------
// MsgBurnNative

// ValidateBasic performs stateless validation checks. Impl sdk.Msg.
func (m MsgBurnNative) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid sender (%s): %s", m.Sender, err)
	}

	if err := validateCoin(m.Coin); err != nil {
		return err
	}

	return nil
}

// GetSigners: Impl sdk.Msg.
func (m MsgBurnNative) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{sender}
}

// Route: Impl legacytx.LegacyMsg. The mesage route must be alphanumeric or empty.
func (m MsgBurnNative) Route() string { return RouterKey }

// Type: Impl legacytx.LegacyMsg. Returns a human-readable string for the message,
// intended for utilization within tags
func (m MsgBurnNative) Type() string { return "burn_native" }

// GetSignBytes: Get the canonical byte representation of the Msg. Impl
// legacytx.LegacyMsg.
func (m MsgBurnNative) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}
