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
	_ sdk.Msg = &MsgAllocateEpochRebates{}
	_ sdk.Msg = &MsgWithdrawEpochRebates{}
	_ sdk.Msg = &MsgShiftPegMultiplier{}
	_ sdk.Msg = &MsgShiftSwapInvariant{}
	_ sdk.Msg = &MsgWithdrawFromPerpFund{}
)

// ------------------------ MsgRemoveMargin ------------------------

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

// ------------------------ MsgAddMargin ------------------------

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

// ------------------------ MsgMarketOrder ------------------------

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

// ------------------------ MsgMultiLiquidate ------------------------

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

// ------------------------ MsgClosePosition ------------------------

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

// ------------------------ MsgSettlePosition ------------------------

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

// ------------------------ MsgDonateToEcosystemFund ------------------------

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

// ------------------------ MsgPartialClose ------------------------

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

// ------------------------ MsgChangeCollateralDenom ------------------------

func (m MsgChangeCollateralDenom) Route() string { return "perp" }
func (m MsgChangeCollateralDenom) Type() string  { return "change_collateral_denom_msg" }

func (m MsgChangeCollateralDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := sdk.ValidateDenom(m.NewDenom); err != nil {
		return err
	}
	return nil
}

func (m MsgChangeCollateralDenom) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgChangeCollateralDenom) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// ------------------------ MsgAllocateEpochRebates ------------------------

func (m MsgAllocateEpochRebates) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Rebates.Validate(); err != nil {
		return err
	}
	return nil
}

func (m MsgAllocateEpochRebates) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgAllocateEpochRebates) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ------------------------ MsgWithdrawEpochRebates ------------------------

func (m MsgWithdrawEpochRebates) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if len(m.Epochs) == 0 {
		return fmt.Errorf("epochs cannot be empty")
	}
	return nil
}

func (m MsgWithdrawEpochRebates) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgWithdrawEpochRebates) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ------------------------ MsgShiftPegMultiplier ------------------------

func (m MsgShiftPegMultiplier) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	if !m.NewPegMult.IsPositive() {
		return fmt.Errorf("%w: got value %s", ErrAmmNonPositivePegMult, m.NewPegMult)
	}
	return nil
}

func (m MsgShiftPegMultiplier) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgShiftPegMultiplier) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ------------------------ MsgShiftSwapInvariant ------------------------

func (m MsgShiftSwapInvariant) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	if !m.NewSwapInvariant.IsPositive() {
		return fmt.Errorf("%w: got value %s", ErrAmmNonPositiveSwapInvariant, m.NewSwapInvariant)
	}
	return nil
}

func (m MsgShiftSwapInvariant) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgShiftSwapInvariant) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ------------------------ MsgCloseMarket ------------------------

func (m MsgCloseMarket) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrapf(errors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if err := m.Pair.Validate(); err != nil {
		return err
	}
	return nil
}

func (m MsgCloseMarket) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// ------------------------ MsgWithdrawFromPerpFund ------------------------

func (m MsgWithdrawFromPerpFund) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("%w: invalid sender address (%s): %s",
			errors.ErrInvalidAddress, m.Sender, err,
		)
	}
	if !m.Amount.IsPositive() {
		return fmt.Errorf(
			"%w: msg \"amount\" must be positive, got %s", ErrGeneric, m.Amount)
	}
	if _, err := sdk.AccAddressFromBech32(m.ToAddr); err != nil {
		return fmt.Errorf("%w: invalid \"to_addr\" (%s): %s",
			errors.ErrInvalidAddress, m.ToAddr, err,
		)
	}
	return nil
}

func (m MsgWithdrawFromPerpFund) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgWithdrawFromPerpFund) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// ------------------------ MsgCreateMarket ------------------------

func (m MsgCreateMarket) ValidateBasic() error {
	if err := m.Pair.Validate(); err != nil {
		return err
	}
	if !m.SqrtDepth.IsPositive() {
		return fmt.Errorf("sqrt depth must be positive, not: %v", m.SqrtDepth.String())
	}
	if !m.PriceMultiplier.IsPositive() {
		return fmt.Errorf("price multiplier must be positive, not: %v", m.PriceMultiplier.String())
	}
	return nil
}

func (m MsgCreateMarket) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}
