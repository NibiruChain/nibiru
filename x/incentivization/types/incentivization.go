package types

import (
	"fmt"
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
