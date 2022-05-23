package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

var _ sdk.Msg = &MsgRemoveMargin{}
var _ sdk.Msg = &MsgAddMargin{}
var _ sdk.Msg = &MsgOpenPosition{}

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

func (m *MsgOpenPosition) ValidateBasic() error {
	if m.Side != Side_SELL && m.Side != Side_BUY {
		return fmt.Errorf("invalid side")
	}
	if _, err := common.NewTokenPairFromStr(m.TokenPair); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}
	if !m.Leverage.GT(sdk.ZeroDec()) {
		return fmt.Errorf("leverage must always be greater than zero")
	}
	if !m.BaseAssetAmountLimit.GT(sdk.ZeroInt()) {
		return fmt.Errorf("base asset amount limit must always be greater than zero")
	}
	if !m.QuoteAssetAmount.GT(sdk.ZeroInt()) {
		return fmt.Errorf("quote asset amount must be always greater than zero")
	}

	return nil
}

func (m *MsgOpenPosition) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{addr}
}
