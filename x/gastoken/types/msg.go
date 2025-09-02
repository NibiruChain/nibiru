package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	gethcommon "github.com/ethereum/go-ethereum/common"
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
	// Validate ERC-20 address
	if !gethcommon.IsHexAddress(m.FeeToken.Erc20Address) {
		return fmt.Errorf("invalid fee_token.erc20_address: %q", m.FeeToken.Erc20Address)
	}
	if gethcommon.HexToAddress(m.FeeToken.Erc20Address) == (gethcommon.Address{}) {
		return fmt.Errorf("fee_token.erc20_address must not be the zero address")
	}
	// Name required only when adding
	if m.Action == FeeTokenUpdateAction_FEE_TOKEN_ACTION_ADD && m.FeeToken.Name == "" {
		return fmt.Errorf("fee_token.name must be set for add action")
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

	if err := m.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}
