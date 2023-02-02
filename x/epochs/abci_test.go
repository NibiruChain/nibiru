package epochs_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/epochs"
	"github.com/NibiruChain/nibiru/x/epochs/keeper"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestEpochInfoChangesBeginBlockerAndInitGenesis(t *testing.T) {
	var app *app.NibiruApp
	var ctx sdk.Context

	now := time.Now()

	tests := []struct {
		expCurrentEpochStartTime   time.Time
		expCurrentEpochStartHeight int64
		expCurrentEpoch            uint64
		expInitialEpochStartTime   time.Time
		fn                         func()
	}{
		{
			// Only advance 2 seconds, do not increment epoch
			expCurrentEpochStartHeight: 2,
			expCurrentEpochStartTime:   now,
			expCurrentEpoch:            1,
			expInitialEpochStartTime:   now,
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			expCurrentEpochStartHeight: 2,
			expCurrentEpochStartTime:   now,
			expCurrentEpoch:            1,
			expInitialEpochStartTime:   now,
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 31))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		// Test that incrementing _exactly_ 1 month increments the epoch count.
		{
			expCurrentEpochStartHeight: 3,
			expCurrentEpochStartTime:   now.Add(time.Hour * 24 * 31),
			expCurrentEpoch:            2,
			expInitialEpochStartTime:   now,
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 32))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			expCurrentEpochStartHeight: 3,
			expCurrentEpochStartTime:   now.Add(time.Hour * 24 * 31),
			expCurrentEpoch:            2,
			expInitialEpochStartTime:   now,
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 32))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx.WithBlockHeight(4).WithBlockTime(now.Add(time.Hour * 24 * 33))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			expCurrentEpochStartHeight: 3,
			expCurrentEpochStartTime:   now.Add(time.Hour * 24 * 31),
			expCurrentEpoch:            2,
			expInitialEpochStartTime:   now,
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(now.Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				ctx = ctx.WithBlockHeight(3).WithBlockTime(now.Add(time.Hour * 24 * 32))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				numBlocksSinceStart, _ := NumBlocksSinceEpochStart(ctx, app.EpochsKeeper, "monthly")
				require.Equal(t, int64(0), numBlocksSinceStart)
				ctx = ctx.WithBlockHeight(4).WithBlockTime(now.Add(time.Hour * 24 * 33))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
				numBlocksSinceStart, _ = NumBlocksSinceEpochStart(ctx, app.EpochsKeeper, "monthly")
				require.Equal(t, int64(1), numBlocksSinceStart)
			},
		},
	}

	for _, test := range tests {
		app, ctx = simapp.NewTestNibiruAppAndContext(true)

		// On init genesis, default epochs information is set
		// To check init genesis again, should make it fresh status
		epochInfos := app.EpochsKeeper.AllEpochInfos(ctx)
		for _, epochInfo := range epochInfos {
			app.EpochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
		}

		ctx = ctx.WithBlockHeight(1).WithBlockTime(now)

		// check init genesis
		epochs.InitGenesis(ctx, app.EpochsKeeper, types.GenesisState{
			Epochs: []types.EpochInfo{
				{
					Identifier:              "monthly",
					StartTime:               ctx.BlockTime(),
					Duration:                time.Hour * 24 * 31,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: ctx.BlockHeight(),
					CurrentEpochStartTime:   ctx.BlockTime(),
					EpochCountingStarted:    false,
				},
			},
		})

		test.fn()

		epochInfo := app.EpochsKeeper.GetEpochInfo(ctx, "monthly")

		require.Equal(t, epochInfo.Identifier, "monthly")
		require.Equal(t, test.expInitialEpochStartTime.UTC().String(), epochInfo.StartTime.UTC().String())
		require.Equal(t, time.Hour*24*31, epochInfo.Duration)
		require.Equal(t, epochInfo.CurrentEpoch, test.expCurrentEpoch)
		require.Equal(t, epochInfo.CurrentEpochStartHeight, test.expCurrentEpochStartHeight)
		require.Equal(t, epochInfo.CurrentEpochStartTime.UTC().String(), test.expCurrentEpochStartTime.UTC().String())
		require.Equal(t, epochInfo.EpochCountingStarted, true)
	}
}

func TestEpochStartingOneMonthAfterInitGenesis(t *testing.T) {
	app, ctx := simapp.NewTestNibiruAppAndContext(true)

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
	app, ctx := simapp.NewTestNibiruAppAndContext(true)
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

// NumBlocksSinceEpochStart returns the number of blocks since the epoch started.
// if the epoch started on block N, then calling this during block N (after BeforeEpochStart)
// would return 0.
// Calling it any point in block N+1 (assuming the epoch doesn't increment) would return 1.
func NumBlocksSinceEpochStart(ctx sdk.Context, k keeper.Keeper, identifier string) (int64, error) {
	epoch := k.GetEpochInfo(ctx, identifier)
	if (epoch == types.EpochInfo{}) {
		return 0, fmt.Errorf("epoch with identifier %s not found", identifier)
	}
	return ctx.BlockHeight() - epoch.CurrentEpochStartHeight, nil
}
