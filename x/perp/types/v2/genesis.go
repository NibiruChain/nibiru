package v2

import (
	"fmt"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Markets:          []Market{},
		Amms:             []AMM{},
		Positions:        []Position{},
		ReserveSnapshots: []ReserveSnapshot{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	for i, pos := range gs.Positions {
		if err := pos.Validate(); err != nil {
			return fmt.Errorf("malformed genesis position %s at index %d: %w", &pos, i, err)
		}
	}

	for i, m := range gs.Markets {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("malformed pair metadata %s at index %d: %w", m, i, err)
		}
	}

	for i, m := range gs.Amms {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("malformed prepaid bad debt %s at index %d: %w", m, i, err)
		}
	}

	return nil
}
