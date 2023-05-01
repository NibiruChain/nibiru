package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestUpsertEpochInfo_HappyPath(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	epochInfo := types.EpochInfo{
		Identifier:              "monthly",
		StartTime:               time.Time{},
		Duration:                time.Hour * 24 * 30,
		CurrentEpoch:            0,
		CurrentEpochStartHeight: 0,
		CurrentEpochStartTime:   time.Time{},
		EpochCountingStarted:    false,
	}

	nibiruApp.EpochsKeeper.UpsertEpochInfo(ctx, epochInfo)
	epochInfoSaved := nibiruApp.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo, epochInfoSaved)

	allEpochs := nibiruApp.EpochsKeeper.AllEpochInfos(ctx)

	require.Len(t, allEpochs, 3)
	// Epochs are ordered in alphabetical order
	require.Equal(t, "15 min", allEpochs[0].Identifier)
	require.Equal(t, "30 min", allEpochs[1].Identifier)
	require.Equal(t, "monthly", allEpochs[2].Identifier)
}

func TestEpochExists(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	epochInfo := types.EpochInfo{
		Identifier:            "monthly",
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 30,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}
	nibiruApp.EpochsKeeper.UpsertEpochInfo(ctx, epochInfo)

	require.True(t, nibiruApp.EpochsKeeper.EpochExists(ctx, "monthly"))
	require.False(t, nibiruApp.EpochsKeeper.EpochExists(ctx, "unexisting-epoch"))
}

func TestItFailsAddingEpochThatExists(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	epochInfo := types.EpochInfo{
		Identifier:            "monthly",
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 30,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}

	err := nibiruApp.EpochsKeeper.AddEpochInfo(ctx, epochInfo)
	require.NoError(t, err)

	// It fails if we try to add it again.
	err = nibiruApp.EpochsKeeper.AddEpochInfo(ctx, epochInfo)
	require.Error(t, err)
}
