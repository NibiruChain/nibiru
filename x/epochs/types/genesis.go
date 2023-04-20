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
	epochs := []EpochInfo{
		{
			Identifier:              WeekEpochID,
			StartTime:               time.Time{},
			Duration:                time.Hour * 24 * 7,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              DayEpochID,
			StartTime:               time.Time{},
			Duration:                time.Hour * 24,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              FifteenMinuteEpochID,
			StartTime:               time.Time{},
			Duration:                time.Second * 60 * 15,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              ThirtyMinuteEpochID,
			StartTime:               time.Time{},
			Duration:                time.Second * 60 * 30,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
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
