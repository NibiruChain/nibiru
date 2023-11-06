package epochs_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestEpochsExportGenesis(t *testing.T) {
	chainStartTime := time.Now().UTC()
	moduleGenesisIn := types.DefaultGenesisFromTime(chainStartTime)

	encCfg := app.MakeEncodingConfig()
	appGenesis := genesis.NewTestGenesisState(encCfg)
	appGenesis[types.ModuleName] = encCfg.Marshaler.MustMarshalJSON(moduleGenesisIn)

	app := testapp.NewNibiruTestApp(appGenesis)
	ctx := testapp.NewContext(app).WithBlockTime(chainStartTime)

	genesis := epochs.ExportGenesis(ctx, app.EpochsKeeper)
	require.Len(t, genesis.Epochs, 4)

	errMsg := fmt.Sprintf("app.EpochsKeeper.AllEpochInfos(ctx): %v\n", app.EpochsKeeper.AllEpochInfos(ctx))
	assert.EqualValues(t, genesis.Epochs[0].Identifier, "30 min")
	assert.EqualValues(t, genesis.Epochs[0].StartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[0].Duration, time.Minute*30, errMsg)
	assert.EqualValues(t, genesis.Epochs[0].CurrentEpoch, uint64(0))
	assert.EqualValues(t, genesis.Epochs[0].CurrentEpochStartHeight, int64(0))
	assert.EqualValues(t, genesis.Epochs[0].CurrentEpochStartTime, chainStartTime)
	assert.EqualValues(t, genesis.Epochs[0].EpochCountingStarted, false)

	assert.EqualValues(t, genesis.Epochs[1].Identifier, "day")
	assert.EqualValues(t, genesis.Epochs[1].StartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[1].Duration, time.Hour*24)
	assert.EqualValues(t, genesis.Epochs[1].CurrentEpoch, uint64(0))
	assert.EqualValues(t, genesis.Epochs[1].CurrentEpochStartHeight, int64(0))
	assert.EqualValues(t, genesis.Epochs[1].CurrentEpochStartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[1].EpochCountingStarted, false)

	assert.EqualValues(t, genesis.Epochs[2].Identifier, "month")
	assert.EqualValues(t, genesis.Epochs[2].StartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[2].Duration, time.Hour*24*30)
	assert.EqualValues(t, genesis.Epochs[2].CurrentEpoch, uint64(0))
	assert.EqualValues(t, genesis.Epochs[2].CurrentEpochStartHeight, int64(0))
	assert.EqualValues(t, genesis.Epochs[2].CurrentEpochStartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[2].EpochCountingStarted, false)

	assert.EqualValues(t, genesis.Epochs[3].Identifier, "week")
	assert.EqualValues(t, genesis.Epochs[3].StartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[3].Duration, time.Hour*24*7)
	assert.EqualValues(t, genesis.Epochs[3].CurrentEpoch, uint64(0))
	assert.EqualValues(t, genesis.Epochs[3].CurrentEpochStartHeight, int64(0))
	assert.EqualValues(t, genesis.Epochs[3].CurrentEpochStartTime, chainStartTime, errMsg)
	assert.EqualValues(t, genesis.Epochs[3].EpochCountingStarted, false)
}

func TestEpochsInitGenesis(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	// On init genesis, default epochs information is set
	// To check init genesis again, should make it fresh status
	epochInfos := app.EpochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range epochInfos {
		err := app.EpochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
		require.NoError(t, err)
	}

	now := time.Now()
	ctx = ctx.WithBlockHeight(1)
	ctx = ctx.WithBlockTime(now)

	// test genesisState validation
	genesisState := types.GenesisState{
		Epochs: []types.EpochInfo{
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
		},
	}
	err := epochs.InitGenesis(ctx, app.EpochsKeeper, genesisState)
	require.Error(t, err)

	assert.EqualError(t, genesisState.Validate(), "epoch identifier should be unique")

	genesisState = types.GenesisState{
		Epochs: []types.EpochInfo{
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
		},
	}

	err = epochs.InitGenesis(ctx, app.EpochsKeeper, genesisState)
	require.NoError(t, err)
	epochInfo, err := app.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.NoError(t, err)
	assert.EqualValues(t, epochInfo.Identifier, "monthly")
	assert.EqualValues(t, epochInfo.StartTime.UTC().String(), now.UTC().String())
	assert.EqualValues(t, epochInfo.Duration, time.Hour*24)
	assert.EqualValues(t, epochInfo.CurrentEpoch, uint64(0))
	assert.EqualValues(t, epochInfo.CurrentEpochStartHeight, ctx.BlockHeight())
	assert.EqualValues(t, epochInfo.CurrentEpochStartTime.UTC().String(), time.Time{}.String())
	assert.EqualValues(t, epochInfo.EpochCountingStarted, true)
}
