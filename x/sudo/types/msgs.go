package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgEditSudoers{}
	_ sdk.Msg = &MsgChangeRoot{}
)

// MsgEditSudoers

func (m MsgEditSudoers) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	for _, contract := range m.Contracts {
		if _, err := sdk.AccAddressFromBech32(contract); err != nil {
			return err
		}
	}

	if !RootActions.Has(m.RootAction()) {
		return fmt.Errorf(
			"invalid action type %s, expected one of %s",
			m.Action, RootActions.ToSlice(),
		)
	}

	return nil
}

func (m MsgEditSudoers) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// Route Implements Msg.
func (msg MsgEditSudoers) Route() string { return ModuleName }

// Type Implements Msg.
func (msg MsgEditSudoers) Type() string { return "edit_sudoers" }

// GetSignBytes Implements Msg.
func (m MsgEditSudoers) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgEditSudoers) RootAction() RootAction {
	return RootAction(m.Action)
}

// MsgChangeRoot

func (m MsgChangeRoot) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m MsgChangeRoot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(m.NewRoot); err != nil {
		return err
	}

	return nil
}

// Route Implements Msg.
func (msg MsgChangeRoot) Route() string { return ModuleName }

// Type Implements Msg.
func (msg MsgChangeRoot) Type() string { return "change_root" }

// GetSignBytes Implements Msg.
func (m MsgChangeRoot) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}
