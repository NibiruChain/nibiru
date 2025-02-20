package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultGenesis(t *testing.T) {
	genState := DefaultGenesis()

	expectedEpochs := []EpochInfo{
		{
			Identifier:              ThirtyMinuteEpochID,
			StartTime:               time.Time{},
			Duration:                30 * time.Minute,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              DayEpochID,
			StartTime:               time.Time{},
			Duration:                24 * time.Hour,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              WeekEpochID,
			StartTime:               time.Time{},
			Duration:                7 * 24 * time.Hour,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
		{
			Identifier:              MonthEpochID,
			StartTime:               time.Time{},
			Duration:                30 * 24 * time.Hour,
			CurrentEpoch:            0,
			CurrentEpochStartHeight: 0,
			CurrentEpochStartTime:   time.Time{},
			EpochCountingStarted:    false,
		},
	}

	// Ensure that genState and expectedEpochs are the same
	require.Equal(t, expectedEpochs, genState.Epochs)
}

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

func TestEpochInfo_HappyPath(t *testing.T) {
	tests := []struct {
		name      string
		epochInfo EpochInfo
	}{
		{
			name: "empty identifier",
			epochInfo: EpochInfo{
				Identifier:              "myEpoch",
				StartTime:               time.Now(),
				Duration:                10 * time.Minute,
				CurrentEpoch:            1,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    false,
				CurrentEpochStartHeight: 1,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.epochInfo.Validate()
			require.NoError(t, err)
		})
	}
}
