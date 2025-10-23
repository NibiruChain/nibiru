package sudo

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var (
	_ legacytx.LegacyMsg = &MsgEditSudoers{}
	_ legacytx.LegacyMsg = &MsgChangeRoot{}
	_ sdk.Msg            = (*MsgEditZeroGasActors)(nil)
)

// ----------------- "nibiru.sudo.v1.MsgEditSudoers" -----------------

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

// GetSigners implements the sdk.Msg interface.
func (m MsgEditSudoers) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// Route implements the sdk.Msg interface.
func (msg MsgEditSudoers) Route() string { return ModuleName }

// Type implements the sdk.Msg interface.
func (msg MsgEditSudoers) Type() string { return "edit_sudoers" }

// GetSignBytes implements the sdk.Msg interface.
func (m MsgEditSudoers) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgEditSudoers) RootAction() RootAction {
	return RootAction(m.Action)
}

// ----------------- "nibiru.sudo.v1.MsgChangeRoot" -----------------

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

// ----------------- "nibiru.sudo.v1.MsgEditZeroGasActors" -----------------

// ValidateBasic performs a stateless validation check. Stateless here means no
// usage of information from the the "world state" [sdk.Context].
func (m MsgEditZeroGasActors) ValidateBasic() error {
	err := m.Actors.Validate()
	if err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("ZeroGasActors stateless validation error: %w", err)
	}

	return nil
}

func (actors ZeroGasActors) Validate() error {
	// Check if each contract is an eligible EVM or Wasm contract address
	for _, contract := range actors.Contracts {
		req := &evm.QueryEthAccountRequest{Address: contract}
		_, err := req.Validate()
		if err != nil {
			return fmt.Errorf("ZeroGasActors stateless validation error: %w", err)
		}
	}

	for _, sender := range actors.Senders {
		_, err := sdk.AccAddressFromBech32(sender)
		if err != nil {
			return fmt.Errorf("ZeroGasActors stateless validation error: %w", err)
		}
	}
	return nil
}

func DefaultZeroGasActors() ZeroGasActors {
	return ZeroGasActors{
		Senders:   []string{},
		Contracts: []string{},
	}
}

// GetSigners returns the addrs of signers that must sign.
// CONTRACT: All signatures must be present to be valid.
// CONTRACT: Returns addrs in some deterministic order.
func (m MsgEditZeroGasActors) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
