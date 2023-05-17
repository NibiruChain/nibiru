package epochs_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestEpochInfoChangesBeginBlockerAndInitGenesis(t *testing.T) {
	var app *app.NibiruApp
	var ctx sdk.Context

	now := time.Now().UTC()

	tests := []struct {
		name              string
		when              func()
		expectedEpochInfo types.EpochInfo
	}{
		{
			name: "no increment",
			when: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
			expectedEpochInfo: types.EpochInfo{
				Identifier:              "monthly",
				StartTime:               now,
				Duration:                time.Hour * 24 * 31,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: 1,
				CurrentEpochStartTime:   now,
				EpochCountingStarted:    true,
			},
		},
		{
			name: "increment",
			when: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 32))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
			expectedEpochInfo: types.EpochInfo{
				Identifier:              "monthly",
				StartTime:               now,
				Duration:                time.Hour * 24 * 31,
				CurrentEpoch:            2,
				CurrentEpochStartHeight: 3,
				CurrentEpochStartTime:   now.Add(time.Hour * 24 * 32),
				EpochCountingStarted:    true,
			},
		},
		// Test that incrementing _exactly_ 1 month increments the epoch count.
		{
			name: "exact increment",
			when: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Hour * 24 * 31))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
			expectedEpochInfo: types.EpochInfo{
				Identifier:              "monthly",
				StartTime:               now,
				Duration:                time.Hour * 24 * 31,
				CurrentEpoch:            2,
				CurrentEpochStartHeight: 2,
				CurrentEpochStartTime:   now.Add(time.Hour * 24 * 31),
				EpochCountingStarted:    true,
			},
		},
		{
			name: "increment twice",
			when: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Hour * 24 * 31))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 31 * 2))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
			expectedEpochInfo: types.EpochInfo{
				Identifier:              "monthly",
				StartTime:               now,
				Duration:                time.Hour * 24 * 31,
				CurrentEpoch:            3,
				CurrentEpochStartHeight: 3,
				CurrentEpochStartTime:   now.Add(time.Hour * 24 * 31 * 2),
				EpochCountingStarted:    true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx = testapp.NewNibiruTestAppAndContext(true)

			// On init genesis, default epochs information is set
			// To check init genesis again, should make it fresh status
			epochInfos := app.EpochsKeeper.AllEpochInfos(ctx)
			for _, epochInfo := range epochInfos {
				app.EpochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
			}

			// insert epoch info that's already begun
			ctx = ctx.WithBlockHeight(1).WithBlockTime(now)
			_ = app.EpochsKeeper.AddEpochInfo(ctx, types.EpochInfo{
				Identifier:              "monthly",
				StartTime:               now,
				Duration:                time.Hour * 24 * 31,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: 1,
				CurrentEpochStartTime:   now,
				EpochCountingStarted:    true,
			})

			tc.when()

			epochInfo := app.EpochsKeeper.GetEpochInfo(ctx, "monthly")
			assert.Equal(t, tc.expectedEpochInfo.CurrentEpoch, epochInfo.CurrentEpoch)
			assert.Equal(t, tc.expectedEpochInfo.CurrentEpochStartTime, epochInfo.CurrentEpochStartTime)
			assert.Equal(t, tc.expectedEpochInfo.CurrentEpochStartHeight, epochInfo.CurrentEpochStartHeight)
		})
	}
}

func TestEpochStartingOneMonthAfterInitGenesis(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	// On init genesis, default epochs information is set
	// To check init genesis again, should make it fresh status
	epochInfos := app.EpochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range epochInfos {
		app.EpochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
	}

	now := time.Now()
	week := time.Hour * 24 * 7
	month := time.Hour * 24 * 30
	initialBlockHeight := int64(1)
	ctx = ctx.WithBlockHeight(initialBlockHeight).WithBlockTime(now)

	epochs.InitGenesis(ctx, app.EpochsKeeper, types.GenesisState{
		Epochs: []types.EpochInfo{
			{
				Identifier:              "monthly",
				StartTime:               now.Add(month),
				Duration:                time.Hour * 24 * 30,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    false,
			},
		},
	})

	// epoch not started yet
	epochInfo := app.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo.CurrentEpoch, uint64(0))
	require.Equal(t, epochInfo.CurrentEpochStartHeight, initialBlockHeight)
	require.Equal(t, epochInfo.CurrentEpochStartTime, time.Time{})
	require.Equal(t, epochInfo.EpochCountingStarted, false)

	// after 1 week
	ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(week))
	epochs.BeginBlocker(ctx, app.EpochsKeeper)

	// epoch not started yet
	epochInfo = app.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo.CurrentEpoch, uint64(0))
	require.Equal(t, epochInfo.CurrentEpochStartHeight, initialBlockHeight)
	require.Equal(t, epochInfo.CurrentEpochStartTime, time.Time{})
	require.Equal(t, epochInfo.EpochCountingStarted, false)

	// after 1 month
	ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(month))
	epochs.BeginBlocker(ctx, app.EpochsKeeper)

	// epoch started
	epochInfo = app.EpochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo.CurrentEpoch, uint64(1))
	require.Equal(t, epochInfo.CurrentEpochStartHeight, ctx.BlockHeight())
	require.Equal(t, epochInfo.CurrentEpochStartTime.UTC().String(), now.Add(month).UTC().String())
	require.Equal(t, epochInfo.EpochCountingStarted, true)
}

// This test ensures legacy EpochInfo messages will not throw errors via InitGenesis and BeginBlocker
func TestLegacyEpochSerialization(t *testing.T) {
	// Legacy Epoch Info message - without CurrentEpochStartHeight property
	legacyEpochInfo := types.EpochInfo{
		Identifier:            "monthly",
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 31,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}

	now := time.Now()
	app, ctx := testapp.NewNibiruTestAppAndContext(true)
	// On init genesis, default epochs information is set
	// To check init genesis again, should make it fresh status
	epochInfos := app.EpochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range epochInfos {
		app.EpochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
	}

	ctx = ctx.WithBlockHeight(1).WithBlockTime(now)

	// check init genesis
	epochs.InitGenesis(ctx, app.EpochsKeeper, types.GenesisState{
		Epochs: []types.EpochInfo{legacyEpochInfo},
	})

	// Do not increment epoch
	ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
	epochs.BeginBlocker(ctx, app.EpochsKeeper)

	// Increment epoch
	ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 32))
	epochs.BeginBlocker(ctx, app.EpochsKeeper)
	epochInfo := app.EpochsKeeper.GetEpochInfo(ctx, "monthly")

	require.NotEqual(t, epochInfo.CurrentEpochStartHeight, int64(0))
}
