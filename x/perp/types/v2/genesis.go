package v2

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Markets:          []Market{},
		Amms:             []AMM{},
		Positions:        []Position{},
		ReserveSnapshots: []ReserveSnapshot{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {

	for _, m := range gs.Markets {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	for _, m := range gs.Amms {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	for _, pos := range gs.Positions {
		if err := pos.Validate(); err != nil {
			return err
		}
	}

	return nil
}
