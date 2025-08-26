package types

import (
	fmt "fmt"

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
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.FeeToken == nil {
		return fmt.Errorf("fee_token must be set")
	}

	if !m.Action.IsValid() {
		return fmt.Errorf("invalid action: must be 0 (add token) or 1 (remove token)")
	}
	return nil
}

func (action FeeTokenUpdateAction) IsValid() bool {
	return action == FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD || action == FeeTokenUpdateAction_FEE_TOKEN_ACTION_REMOVE
}

const TypeMsgUpdateParams = "update_params"

// Route implements legacytx.LegacyMsg
func (msg MsgUpdateParams) Route() string { return RouterKey }

func (msg MsgUpdateParams) Type() string { return TypeMsgUpdateParams }

func (msg MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(fmt.Errorf("invalid sender address: %w", err))
	}
	return []sdk.AccAddress{addr}
}

func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}
