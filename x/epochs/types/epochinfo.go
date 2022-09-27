package types

import (
	"errors"
	"time"
)

func NewEpochInfo(identifier string) EpochInfo {
	return EpochInfo{
		Identifier:              identifier,
		StartTime:               time.Time{},
		Duration:                0,
		CurrentEpoch:            0,
		CurrentEpochStartTime:   time.Time{},
		EpochCountingStarted:    false,
		CurrentEpochStartHeight: 0,
	}
}

func (e *EpochInfo) Initialize() {

}

// Validate also validates epoch info.
func (e *EpochInfo) Validate() error {
	if e.Identifier == "" {
		return errors.New("epoch identifier should NOT be empty")
	}

	if e.Duration == 0 {
		return errors.New("epoch duration should NOT be 0")
	}

	if e.CurrentEpochStartHeight < 0 {
		return errors.New("epoch CurrentEpoch Start Height must be non-negative")
	}

	return nil
}
