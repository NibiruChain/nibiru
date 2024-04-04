package types

import (
	sdkmath "cosmossdk.io/math"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

// ensure Msg interface compliance at compile time
var (
	_ legacytx.LegacyMsg = &MsgEditInflationParams{}
	_ legacytx.LegacyMsg = &MsgToggleInflation{}
	_ legacytx.LegacyMsg = &MsgBurn{}
)

// oracle message types
const (
	TypeMsgEditInflationParams = "edit_inflation_params"
	TypeMsgToggleInflation     = "toggle_inflation"
	TypeMsgBurn                = "msg_burn"
)

// Route implements legacytx.LegacyMsg
func (msg MsgEditInflationParams) Route() string { return RouterKey }

// Type implements legacytx.LegacyMsg
func (msg MsgEditInflationParams) Type() string { return TypeMsgEditInflationParams }

// GetSignBytes implements legacytx.LegacyMsg
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

	if m.InflationDistribution != nil {
		if m.InflationDistribution.CommunityPool.IsNil() {
			return fmt.Errorf("inflation distribution community pool should not be nil")
		}
		if m.InflationDistribution.StakingRewards.IsNil() {
			return fmt.Errorf("inflation distribution staking rewards should not be nil")
		}
		if m.InflationDistribution.StrategicReserves.IsNil() {
			return fmt.Errorf("inflation distribution strategic reserves should not be nil")
		}

		sum := sdkmath.LegacyNewDec(0)
		sum = sum.Add(m.InflationDistribution.CommunityPool)
		sum = sum.Add(m.InflationDistribution.StakingRewards)
		sum = sum.Add(m.InflationDistribution.StrategicReserves)
		if !sum.Equal(sdkmath.LegacyOneDec()) {
			return fmt.Errorf("inflation distribution sum should be 1, got %s", sum)
		}
	}

	if m.PolynomialFactors != nil {
		if len(m.PolynomialFactors) != 6 {
			return fmt.Errorf("polynomial factors should have 6 elements, got %d", len(m.PolynomialFactors))
		}
	}

	return nil
}

// -------------------------------------------------
// MsgToggleInflation
// Route implements legacytx.LegacyMsg
func (msg MsgToggleInflation) Route() string { return RouterKey }

// Type implements legacytx.LegacyMsg
func (msg MsgToggleInflation) Type() string { return TypeMsgToggleInflation }

// GetSignBytes implements legacytx.LegacyMsg
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

// -------------------------------------------------
// MsgBurn
// Route implements legacytx.LegacyMsg
func (msg MsgBurn) Route() string { return RouterKey }

// Type implements legacytx.LegacyMsg
func (msg MsgBurn) Type() string { return TypeMsgBurn }

// GetSignBytes implements legacytx.LegacyMsg
func (msg MsgBurn) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners implements legacytx.LegacyMsg
func (msg MsgBurn) GetSigners() []sdk.AccAddress {
	feeder, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{feeder}
}

func (m MsgBurn) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	if err := m.Coin.Validate(); err != nil {
		return err
	}

	if m.Coin.Amount.IsZero() {
		return fmt.Errorf("coin amount should not be zero")
	}

	return nil
}
