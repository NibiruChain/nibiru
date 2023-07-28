package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name      string
		genState  GenesisState
		errString string
	}{
		{
			name: "duplicate epochinfo",
			genState: GenesisState{
				Epochs: []EpochInfo{
					{
						Identifier:              "repeated",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
					{
						Identifier:              "someOther",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
					{
						Identifier:              "repeated",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
				},
			},
			errString: "epoch identifier should be unique",
		},
		{
			name: "at least one invalid epochinfo",
			genState: GenesisState{
				Epochs: []EpochInfo{
					{
						Identifier:              "repeated",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
					{
						Identifier:              "someOther",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
					{
						Identifier:              "the invalid",
						StartTime:               time.Now(),
						Duration:                0,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
				},
			},
			errString: "epoch duration should NOT be 0",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genState.Validate()
			require.ErrorContains(t, err, tc.errString)
		})
	}
}

func TestGenesisState_HappyPath(t *testing.T) {
	tests := []struct {
		name     string
		genState GenesisState
	}{
		{
			name: "duplicate epochinfo",
			genState: GenesisState{
				Epochs: []EpochInfo{
					{
						Identifier:              "firstOne",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
					{
						Identifier:              "someOther",
						StartTime:               time.Now(),
						Duration:                10,
						CurrentEpoch:            2,
						CurrentEpochStartTime:   time.Now(),
						EpochCountingStarted:    false,
						CurrentEpochStartHeight: 1,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genState.Validate()
			require.NoError(t, err)
		})
	}
}
