package types

import (
	"errors"
)

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
