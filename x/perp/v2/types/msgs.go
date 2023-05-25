package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRemoveMargin{}
var _ sdk.Msg = &MsgAddMargin{}
var _ sdk.Msg = &MsgOpenPosition{}
var _ sdk.Msg = &MsgClosePosition{}
var _ sdk.Msg = &MsgMultiLiquidate{}

// MsgRemoveMargin

func (m MsgRemoveMargin) Route() string { return "perp" }
func (m MsgRemoveMargin) Type() string  { return "remove_margin_msg" }

func (m MsgRemoveMargin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	err := m.Pair.Validate()
	if err != nil {
		return err
	}

	if !m.Margin.Amount.IsPositive() {
		return fmt.Errorf("margin must be positive, not: %v", m.Margin.Amount.String())
	}

	if m.Margin.Denom != m.Pair.QuoteDenom() {
		return fmt.Errorf("invalid margin denom, expected %s, got %s", m.Pair.QuoteDenom(), m.Margin.Denom)
	}

	return nil
}

func (m MsgRemoveMargin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgRemoveMargin) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// MsgAddMargin

func (m MsgAddMargin) Route() string { return "perp" }
func (m MsgAddMargin) Type() string  { return "add_margin_msg" }

func (m MsgAddMargin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	err := m.Pair.Validate()
	if err != nil {
		return err
	}

	if !m.Margin.Amount.IsPositive() {
		return fmt.Errorf("margin must be positive, not: %v", m.Margin.Amount.String())
	}

	if m.Margin.Denom != m.Pair.QuoteDenom() {
		return fmt.Errorf("invalid margin denom, expected %s, got %s", m.Pair.QuoteDenom(), m.Margin.Denom)
	}

	return nil
}

func (m MsgAddMargin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgAddMargin) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// MsgOpenPosition

func (m MsgOpenPosition) Route() string { return "perp" }
func (m MsgOpenPosition) Type() string  { return "open_position_msg" }

func (m *MsgOpenPosition) ValidateBasic() error {
	if m.Side != Direction_SHORT && m.Side != Direction_LONG {
		return fmt.Errorf("invalid side")
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}
	if !m.Leverage.IsPositive() {
		return fmt.Errorf("leverage must always be greater than zero")
	}
	if m.BaseAssetAmountLimit.IsNegative() {
		return fmt.Errorf("base asset amount limit must not be negative")
	}
	if !m.QuoteAssetAmount.IsPositive() {
		return fmt.Errorf("quote asset amount must be always greater than zero")
	}

	return nil
}

func (m MsgOpenPosition) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m *MsgOpenPosition) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// MsgMultiLiquidate

func (m *MsgMultiLiquidate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	for i, liquidation := range m.Liquidations {
		if _, err := sdk.AccAddressFromBech32(liquidation.Trader); err != nil {
			return fmt.Errorf("invalid liquidation at index %d: %w", i, err)
		}

		if err := liquidation.Pair.Validate(); err != nil {
			return fmt.Errorf("invalid liquidation at index %d: %w", i, err)
		}
	}

	return nil
}

func (m *MsgMultiLiquidate) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{addr}
}

// MsgClosePosition

func (m MsgClosePosition) Route() string { return "perp" }
func (m MsgClosePosition) Type() string  { return "close_position_msg" }

func (m MsgClosePosition) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	return nil
}

func (m MsgClosePosition) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgClosePosition) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// MsgDonateToEcosystemFund

func (m MsgDonateToEcosystemFund) Route() string { return "perp" }
func (m MsgDonateToEcosystemFund) Type() string  { return "donate_to_ef_msg" }

func (m MsgDonateToEcosystemFund) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if m.Donation.IsNil() || m.Donation.IsNegative() {
		return fmt.Errorf("invalid donation amount: %s", m.Donation.String())
	}
	return nil
}

func (m MsgDonateToEcosystemFund) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgDonateToEcosystemFund) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
