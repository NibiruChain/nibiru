package types

import (
	"fmt"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

func DefaultGenesis() *GenesisState {
	return new(GenesisState)
}

func (m *GenesisState) Validate() error {
	for _, program := range m.IncentivizationPrograms {
		if program.LpDenom == "" {
			// TODO(mercilex): maybe check valid denom
			return fmt.Errorf("program with ID %d does not have a LP denom set: %s", program.Id, program)
		}
		if program.EscrowAddress == "" {
			// TODO(mercilex): maybe check if valid account address
			return fmt.Errorf("program with ID %d does not have escrow address set: %s", program.Id, program)
		}

		if program.RemainingEpochs == 0 {
			return fmt.Errorf("program with ID %d does not have any remaining epochs: %s", program.Id, program)
		}

		if program.MinLockupDuration == 0 {
			return fmt.Errorf("program with ID %d does not have any lockup duration specified: %s", program.Id, program)
		}
	}

	return nil
}

// msg impl

var (
	_ sdk.Msg = (*MsgCreateIncentivizationProgram)(nil)
	_ sdk.Msg = (*MsgFundIncentivizationProgram)(nil)
)

func (m *MsgCreateIncentivizationProgram) ValidateBasic() error {
	if m.StartTime != nil && *m.StartTime == (time.Time{}) {
		return fmt.Errorf("invalid time")
	}
	if m.LpDenom == "" {
		return fmt.Errorf("invalid denom")
	}
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	if m.Epochs == 0 {
		return fmt.Errorf("invalid epochs")
	}

	if m.MinLockupDuration == nil || *m.MinLockupDuration == 0 {
		return fmt.Errorf("invalid duration")
	}

	if err := m.InitialFunds.Validate(); err != nil {
		return fmt.Errorf("invalid initial funds")
	}
	return nil
}

func (m *MsgCreateIncentivizationProgram) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{addr}
}

func (m *MsgFundIncentivizationProgram) ValidateBasic() error {
	if err := m.Funds.Validate(); err != nil {
		return err
	}
	if m.Funds.IsZero() {
		return fmt.Errorf("no funding provided")
	}

	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return err
	}

	return nil
}

func (m *MsgFundIncentivizationProgram) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{addr}
}

// codec bs

func RegisterInterfaces(r codectypes.InterfaceRegistry) {
	r.RegisterImplementations((*sdk.Msg)(nil), &MsgFundIncentivizationProgram{}, &MsgCreateIncentivizationProgram{})
}
