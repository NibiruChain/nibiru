package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/errors"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgRemoveMargin{}
	_ sdk.Msg = &MsgAddMargin{}
	_ sdk.Msg = &MsgMarketOrder{}
	_ sdk.Msg = &MsgMultiLiquidate{}
	_ sdk.Msg = &MsgClosePosition{}
	_ sdk.Msg = &MsgDonateToEcosystemFund{}
	_ sdk.Msg = &MsgPartialClose{}
)

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

// MsgMarketOrder

func (m MsgMarketOrder) Route() string { return "perp" }
func (m MsgMarketOrder) Type() string  { return "market_order_msg" }

func (m *MsgMarketOrder) ValidateBasic() error {
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

func (m MsgMarketOrder) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m *MsgMarketOrder) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// MsgMultiLiquidate

func (m MsgMultiLiquidate) Route() string { return "perp" }
func (m MsgMultiLiquidate) Type() string  { return "multi_liquidate_msg" }

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

func (m MsgMultiLiquidate) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
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

// MsgSettlePosition

func (m MsgSettlePosition) Route() string { return "perp" }
func (m MsgSettlePosition) Type() string  { return "settle_position_msg" }

func (m MsgSettlePosition) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}

	return nil
}

func (m MsgSettlePosition) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgSettlePosition) GetSigners() []sdk.AccAddress {
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

// MsgPartialClose

func (m MsgPartialClose) Route() string { return "perp" }
func (m MsgPartialClose) Type() string  { return "partial_close_msg" }

func (m MsgPartialClose) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	if !m.Size_.IsPositive() {
		return fmt.Errorf("invalid size amount: %s", m.Size_.String())
	}
	return nil
}

func (m MsgPartialClose) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgPartialClose) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
