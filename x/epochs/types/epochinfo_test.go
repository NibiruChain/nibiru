package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEpochInfo_Validate(t *testing.T) {
	tests := []struct {
		name      string
		epochInfo EpochInfo
		errString string
	}{
		{
			name: "empty identifier",
			epochInfo: EpochInfo{
				Identifier:              "",
				StartTime:               time.Now(),
				Duration:                10 * time.Minute,
				CurrentEpoch:            1,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    false,
				CurrentEpochStartHeight: 1,
			},
			errString: "epoch identifier should NOT be empty",
		},
		{
			name: "epoch duration 0",
			epochInfo: EpochInfo{
				Identifier:              "monthly",
				StartTime:               time.Now(),
				Duration:                0,
				CurrentEpoch:            1,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    false,
				CurrentEpochStartHeight: 1,
			},
			errString: "epoch duration should NOT be 0",
		},
		{
			name: "current epoch start height negative",
			epochInfo: EpochInfo{
				Identifier:              "monthly",
				StartTime:               time.Now(),
				Duration:                10 * time.Minute,
				CurrentEpoch:            10,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    false,
				CurrentEpochStartHeight: -1,
			},
			errString: "epoch CurrentEpoch Start Height must be non-negative",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.epochInfo.Validate()
			require.Error(t, err)
			require.ErrorContains(t, err, tc.errString)
		})
	}
}
