package pb

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgEditSudoers{}

const (
	ModuleName = "sudo"
)

var (
	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	MemStoreKey = "mem_" + ModuleName

	// RouterKey is the message route for transactions.
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key.
	QuerierRoute = ModuleName
)

func (gen GenesisState) Validate() error {
	if gen.Sudoers.Contracts != nil {
		return nil
	} else {
		return fmt.Errorf("nil contract state must be []string")
	}
}

// MsgEditSudoers

func (m MsgEditSudoers) Route() string { return RouterKey }
func (m MsgEditSudoers) Type() string  { return "msg_edit_sudoers" }

func (m MsgEditSudoers) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	for _, contract := range m.Contracts {
		if _, err := sdk.AccAddressFromBech32(contract); err != nil {
			return err
		}
	}

	if !ROOT_ACTIONS.Has(m.Action) {
		return fmt.Errorf(
			"invalid action type %s, expected one of %s",
			m.Action, ROOT_ACTIONS.ToSlice(),
		)
	}

	return nil
}

func (m MsgEditSudoers) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

func (m MsgEditSudoers) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
