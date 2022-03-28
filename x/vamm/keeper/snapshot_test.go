package keeper

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/types/time"

	ammtypes "github.com/MatrixDao/matrix/x/vamm/types"
)

func TestKeeper_saveOrGetReserveSnapshotFailsIfNotSnapshotSavedBefore(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.addReserveSnapshot(ctx, getSamplePool())
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)

	_, err = ammKeeper.getLastReserveSnapshot(ctx, UsdmPair)
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)
}

func TestKeeper_SaveReserveSnapshot(t *testing.T) {
	expectedTime := time.Now()
	expectedBlockHeight := 123
	pool := getSamplePool()

	expectedSnapshot := ammtypes.ReserveSnapshot{
		QuoteAssetReserve: pool.QuoteAssetReserve,
		BaseAssetReserve:  pool.BaseAssetReserve,
		Timestamp:         expectedTime.Unix(),
		BlockNumber:       int64(expectedBlockHeight),
	}

	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(expectedBlockHeight)).WithBlockTime(expectedTime)

	err := ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	snapshot, err := ammKeeper.getLastReserveSnapshot(ctx, UsdmPair)
	require.NoError(t, err)

	require.Equal(t, expectedSnapshot, snapshot)
}

func TestKeeper_saveReserveSnapshot_IncrementsCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(123)).WithBlockTime(time.Now())

	pool := getSamplePool()

	err := ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	// Save another one, counter should be incremented to 2
	err = ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 2)
}

func TestKeeper_updateSnapshot_doesNotIncrementCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(123)).WithBlockTime(time.Now())

	pool := getSamplePool()

	err := ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	// update the snapshot, counter should not be incremented
	pool.QuoteAssetReserve = "20000"
	err = ammKeeper.updateSnapshot(ctx, pool)
	require.NoError(t, err)

	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)

	savedSnap, err := ammKeeper.getLastReserveSnapshot(ctx, pool.Pair)
	require.NoError(t, err)
	require.Equal(t, pool.QuoteAssetReserve, savedSnap.QuoteAssetReserve)
}

func TestNewKeeper_getSnapshotByCounter(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	expectedHeight := int64(123)
	expectedTime := time.Now()

	ctx = ctx.WithBlockHeight(expectedHeight).WithBlockTime(expectedTime)

	pool := getSamplePool()

	expectedSnapshot := ammtypes.ReserveSnapshot{
		QuoteAssetReserve: pool.QuoteAssetReserve,
		BaseAssetReserve:  pool.BaseAssetReserve,
		Timestamp:         expectedTime.Unix(),
		BlockNumber:       expectedHeight,
	}

	err := ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	// Last counter updated
	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 1)
	snapshot, err := ammKeeper.getLastReserveSnapshot(ctx, pool.Pair)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, snapshot)

	// We save another different snapshot
	differentSnapshot := expectedSnapshot
	differentSnapshot.QuoteAssetReserve = "12341234"
	pool.QuoteAssetReserve = differentSnapshot.QuoteAssetReserve
	err = ammKeeper.saveReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	// We get the snapshot 1
	savedSnap, err := ammKeeper.getSnapshotByCounter(ctx, pool.Pair, 1)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, savedSnap)
	require.NotEqual(t, differentSnapshot, snapshot)
}

func requireLastSnapshotCounterEqual(t *testing.T, ctx sdktypes.Context, keeper Keeper, pool *ammtypes.Pool, counter int64) {
	c, found := keeper.getSnapshotCounter(ctx, pool.Pair)
	require.True(t, found)
	require.Equal(t, counter, c)
}
