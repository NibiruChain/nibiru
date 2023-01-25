package vpool_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/vpool"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestSnapshotUpdates(t *testing.T) {
	nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
	vpoolKeeper := nibiruApp.VpoolKeeper

	runBlock := func(duration time.Duration) {
		vpool.EndBlocker(ctx, nibiruApp.VpoolKeeper)
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
	}

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC)).WithBlockHeight(1)

	require.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
		sdk.NewDec(1_000),
		sdk.NewDec(1_000),
		types.DefaultVpoolConfig().
			WithTradeLimitRatio(sdk.OneDec()).
			WithFluctuationLimitRatio(sdk.OneDec()),
	))
	expectedSnapshot := types.NewReserveSnapshot(
		common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
		sdk.NewDec(1_000),
		sdk.NewDec(1_000),
		ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err := vpoolKeeper.ReserveSnapshots.Get(ctx, collections.Join(common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	t.Log("affect mark price")
	baseAmtAbs, err := vpoolKeeper.SwapQuoteForBase(
		ctx,
		common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
		types.Direction_ADD_TO_POOL,
		sdk.NewDec(250), // ← dyAmm
		sdk.ZeroDec(),
		false,
	)
	// dxAmm := (k / (y + dyAmm)) - x = (1e6 / (1e3 + 250)) - 1e3 = -200
	assert.EqualValues(t, sdk.NewDec(200), baseAmtAbs)
	require.NoError(t, err)
	expectedSnapshot = types.NewReserveSnapshot(
		common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
		sdk.NewDec(800),   // ← x + dxAmm
		sdk.NewDec(1_250), // ← y + dyAMM
		ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	ctxAtSnapshot := sdk.Context(ctx) // manually copy ctx before the time skip
	timeSkipDuration := 5 * time.Second
	runBlock(timeSkipDuration) // increments ctx.blockHeight and ctx.BlockTime
	snapshot, err = vpoolKeeper.ReserveSnapshots.Get(ctx,
		collections.Join(common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	testutil.RequireContainsTypedEvent(t, ctx, &types.ReserveSnapshotSavedEvent{
		Pair:           expectedSnapshot.Pair,
		QuoteReserve:   expectedSnapshot.QuoteAssetReserve,
		BaseReserve:    expectedSnapshot.BaseAssetReserve,
		MarkPrice:      snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve),
		BlockHeight:    ctxAtSnapshot.BlockHeight(),
		BlockTimestamp: ctxAtSnapshot.BlockTime(),
	})
}
