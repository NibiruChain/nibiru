package types

import (
	"errors"
	"time"
)

func NewGenesisState(epochs []EpochInfo) *GenesisState {
	return &GenesisState{Epochs: epochs}
}

// DefaultGenesis returns the default Capability genesis state.
func DefaultGenesis() *GenesisState {
	startTime := time.Time{}
	return DefaultGenesisFromTime(startTime)
}

func DefaultGenesisFromTime(startTime time.Time) *GenesisState {
	epochs := []EpochInfo{
		{
			Identifier:              ThirtyMinuteEpochID,
			StartTime:               startTime,
			Duration:                30 * time.Minute,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   startTime,
			EpochCountingStarted:    false,
		},
		{
			Identifier:              DayEpochID,
			StartTime:               startTime,
			Duration:                24 * time.Hour,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   startTime,
			EpochCountingStarted:    false,
		},
		{
			Identifier:              WeekEpochID,
			StartTime:               startTime,
			Duration:                7 * 24 * time.Hour,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   startTime,
			EpochCountingStarted:    false,
		},
	}

	return NewGenesisState(epochs)
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	epochIdentifiers := map[string]bool{}
	for _, epoch := range gs.Epochs {
		if epochIdentifiers[epoch.Identifier] {
			return errors.New("epoch identifier should be unique")
		}

		if err := epoch.Validate(); err != nil {
			return err
		}

		epochIdentifiers[epoch.Identifier] = true
	}

	return nil
}
