package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgEditSudoers{}

// MsgEditSudoers

func (m *MsgEditSudoers) ValidateBasic() error {
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

func (m *MsgEditSudoers) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m *MsgEditSudoers) RootAction() RootAction {
	return RootAction(m.Action)
}

// MsgChangeRoot

func (m *MsgChangeRoot) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (m *MsgChangeRoot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(m.NewRoot); err != nil {
		return err
	}

	return nil
}
