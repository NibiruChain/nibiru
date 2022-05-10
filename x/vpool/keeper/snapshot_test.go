package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/types/time"
)

func TestKeeper_saveOrGetReserveSnapshotFailsIfNotSnapshotSavedBefore(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.addReserveSnapshot(ctx, getSamplePool())
	require.Error(t, err, types.ErrNoLastSnapshotSaved)

	_, _, err = ammKeeper.getLatestReserveSnapshot(ctx, NUSDPair)
	require.Error(t, err, types.ErrNoLastSnapshotSaved)
}

func TestKeeper_SaveSnapshot(t *testing.T) {
	expectedTime := time.Now()
	expectedBlockHeight := 123
	pool := getSamplePool()

	expectedSnapshot := types.ReserveSnapshot{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
		Timestamp:         expectedTime.Unix(),
		BlockNumber:       int64(expectedBlockHeight),
	}

	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(expectedBlockHeight)).WithBlockTime(expectedTime)
	ammKeeper.saveSnapshot(ctx, pool, 0)
	ammKeeper.saveSnapshotCounter(ctx, common.TokenPair(pool.Pair), 0)

	snapshot, counter, err := ammKeeper.getLatestReserveSnapshot(ctx, NUSDPair)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, snapshot)
	require.Equal(t, uint64(0), counter)
}

func TestNewKeeper_getSnapshot(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)
	expectedHeight := int64(123)
	expectedTime := time.Now()

	ctx = ctx.WithBlockHeight(expectedHeight).WithBlockTime(expectedTime)

	pool := getSamplePool()

	firstSnapshot := types.ReserveSnapshot{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
		Timestamp:         expectedTime.Unix(),
		BlockNumber:       expectedHeight,
	}

	t.Log("Save snapshot 0")
	ammKeeper.saveSnapshot(ctx, pool, 0)
	ammKeeper.saveSnapshotCounter(ctx, common.TokenPair(pool.Pair), 0)

	t.Log("Check snapshot 0")
	requireLastSnapshotCounterEqual(t, ctx, ammKeeper, pool, 0)
	oldSnapshot, counter, err := ammKeeper.getLatestReserveSnapshot(ctx, common.TokenPair(pool.Pair))
	require.NoError(t, err)
	require.Equal(t, firstSnapshot, oldSnapshot)
	require.Equal(t, uint64(0), counter)

	t.Log("We save another different snapshot")
	differentSnapshot := firstSnapshot
	differentSnapshot.BaseAssetReserve = sdk.NewIntFromUint64(12341234)
	pool.BaseAssetReserve = differentSnapshot.BaseAssetReserve
	ammKeeper.saveSnapshot(ctx, pool, 1)

	t.Log("Fetch snapshot 1")
	newSnapshot, err := ammKeeper.getSnapshot(ctx, common.TokenPair(pool.Pair), 1)
	require.NoError(t, err)
	require.Equal(t, differentSnapshot, newSnapshot)
	require.NotEqual(t, differentSnapshot, oldSnapshot)
}

func requireLastSnapshotCounterEqual(t *testing.T, ctx sdk.Context, keeper Keeper, pool *types.Pool, counter uint64) {
	c, found := keeper.getSnapshotCounter(ctx, common.TokenPair(pool.Pair))
	require.True(t, found)
	require.Equal(t, counter, c)
}
