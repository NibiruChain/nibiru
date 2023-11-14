package types

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	period uint64,
	skippedEpochs uint64,
) GenesisState {
	return GenesisState{
		Params:        params,
		Period:        period,
		SkippedEpochs: skippedEpochs,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		Period:        0,
		SkippedEpochs: 0,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := validateUint64(gs.SkippedEpochs); err != nil {
		return err
	}

	if err := validateUint64(gs.Period); err != nil {
		return err
	}

	return gs.Params.Validate()
}
