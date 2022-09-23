package keeper_test

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestEpochLifeCycle(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)

	epochInfo := types.EpochInfo{
		Identifier:            "monthly",
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 30,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}

	nibiruApp.EpochsKeeper.SetEpochInfo(ctx, epochInfo)
	epochInfoSaved := nibiruApp.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo, epochInfoSaved)

	allEpochs := nibiruApp.EpochsKeeper.AllEpochInfos(ctx)

	require.Len(t, allEpochs, 5)
	// Epochs are ordered in alphabetical order
	require.Equal(t, "15 min", allEpochs[0].Identifier)
	require.Equal(t, "30 min", allEpochs[1].Identifier)
	require.Equal(t, "day", allEpochs[2].Identifier)
	require.Equal(t, "monthly", allEpochs[3].Identifier)
	require.Equal(t, "week", allEpochs[4].Identifier)
}
