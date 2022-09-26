package types

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genState.Validate()
			require.ErrorContains(t, err, tc.errString)
		})
	}
}
