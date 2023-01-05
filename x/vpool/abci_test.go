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

	assert.NoError(t, vpoolKeeper.CreatePool(
		ctx,
		common.Pair_BTC_NUSD,
		sdk.NewDec(10),
		sdk.NewDec(10),
		types.VpoolConfig{
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.5"),
			FluctuationLimitRatio:  sdk.OneDec(),
			MaxOracleSpreadRatio:   sdk.OneDec(),
			MaintenanceMarginRatio: sdk.OneDec(),
			MaxLeverage:            sdk.NewDec(10),
		},
	))
	expectedSnapshot := types.NewReserveSnapshot(
		common.Pair_BTC_NUSD,
		sdk.NewDec(10),
		sdk.NewDec(10),
		ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err := vpoolKeeper.ReserveSnapshots.Get(ctx, collections.Join(common.Pair_BTC_NUSD, time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	t.Log("affect mark price")
	_, err = vpoolKeeper.SwapQuoteForBase(
		ctx,
		common.Pair_BTC_NUSD,
		types.Direction_ADD_TO_POOL,
		sdk.NewDec(10),
		sdk.ZeroDec(),
		false,
	)
	require.NoError(t, err)
	expectedSnapshot = types.NewReserveSnapshot(
		common.Pair_BTC_NUSD,
		sdk.NewDec(5),
		sdk.NewDec(20),
		ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	ctxAtSnapshot := sdk.Context(ctx) // manually copy ctx before the time skip
	timeSkipDuration := 5 * time.Second
	runBlock(timeSkipDuration) // increments ctx.blockHeight and ctx.BlockTime
	snapshot, err = vpoolKeeper.ReserveSnapshots.Get(ctx, collections.Join(common.Pair_BTC_NUSD, time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	testutil.RequireContainsTypedEvent(t, ctx, &types.ReserveSnapshotSavedEvent{
		Pair:           expectedSnapshot.Pair.String(),
		QuoteReserve:   expectedSnapshot.QuoteAssetReserve,
		BaseReserve:    expectedSnapshot.BaseAssetReserve,
		MarkPrice:      snapshot.QuoteAssetReserve.Quo(snapshot.BaseAssetReserve),
		BlockHeight:    ctxAtSnapshot.BlockHeight(),
		BlockTimestamp: ctxAtSnapshot.BlockTime(),
	})
}
