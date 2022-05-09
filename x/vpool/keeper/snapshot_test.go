package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/types/time"

	ammtypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestKeeper_saveOrGetReserveSnapshotFailsIfNotSnapshotSavedBefore(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.addReserveSnapshot(ctx, getSamplePool())
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)

	_, _, err = ammKeeper.getLastReserveSnapshot(ctx, NUSDPair)
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)
}

func TestKeeper_SaveReserveSnapshot(t *testing.T) {
	expectedTime := time.Now()
	expectedBlockHeight := 123
	pool := getSamplePool()

	expectedSnapshot := ammtypes.ReserveSnapshot{
		Token0Reserve: pool.Token0Reserve,
		Token1Reserve: pool.Token1Reserve,
		Timestamp:     expectedTime.Unix(),
		BlockNumber:   int64(expectedBlockHeight),
	}

	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(expectedBlockHeight)).WithBlockTime(expectedTime)

	err := ammKeeper.saveReserveSnapshot(ctx, 1, pool)
	require.NoError(t, err)

	snapshot, _, err := ammKeeper.getLastReserveSnapshot(ctx, NUSDPair)
	require.NoError(t, err)

	require.Equal(t, expectedSnapshot, snapshot)
}

func TestKeeper_saveReserveSnapshot_IncrementsCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(123)).WithBlockTime(time.Now())

	pool := getSamplePool()

	err := ammKeeper.saveReserveSnapshot(ctx, 0, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	// Save another one, counter should be incremented to 2
	err = ammKeeper.saveReserveSnapshot(ctx, 1, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 2)
}

func TestKeeper_updateSnapshot_doesNotIncrementCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(123)).WithBlockTime(time.Now())

	pool := getSamplePool()

	err := ammKeeper.saveReserveSnapshot(ctx, 0, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	// update the snapshot, counter should not be incremented
	pool.Token0Reserve = "20000"
	err = ammKeeper.updateSnapshot(ctx, 1, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	savedSnap, _, err := ammKeeper.getLastReserveSnapshot(ctx, common.TokenPair(pool.Pair))
	require.NoError(t, err)
	require.Equal(t, pool.Token0Reserve, savedSnap.Token0Reserve)
}

func TestNewKeeper_getSnapshotByCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	expectedHeight := int64(123)
	expectedTime := time.Now()

	ctx = ctx.WithBlockHeight(expectedHeight).WithBlockTime(expectedTime)

	pool := getSamplePool()

	expectedSnapshot := ammtypes.ReserveSnapshot{
		Token0Reserve: pool.Token0Reserve,
		Token1Reserve: pool.Token1Reserve,
		Timestamp:     expectedTime.Unix(),
		BlockNumber:   expectedHeight,
	}

	err := ammKeeper.saveReserveSnapshot(ctx, 0, pool)
	require.NoError(t, err)

	// Last counter updated
	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)
	snapshot, counter, err := ammKeeper.getLastReserveSnapshot(ctx, common.TokenPair(pool.Pair))
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, snapshot)

	// We save another different snapshot
	differentSnapshot := expectedSnapshot
	differentSnapshot.Token0Reserve = "12341234"
	pool.Token0Reserve = differentSnapshot.Token0Reserve
	err = ammKeeper.saveReserveSnapshot(ctx, counter, pool)
	require.NoError(t, err)

	// We get the snapshot 1
	savedSnap, err := ammKeeper.getSnapshotByCounter(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, savedSnap)
	require.NotEqual(t, differentSnapshot, snapshot)
}

func requireLastSnapshotCounterEqual(t *testing.T, ctx sdktypes.Context, keeper Keeper, pool *ammtypes.Pool, counter int64) {
	c, found := keeper.getSnapshotCounter(ctx, common.TokenPair(pool.Pair))
	require.True(t, found)
	require.Equal(t, counter, c)
}
