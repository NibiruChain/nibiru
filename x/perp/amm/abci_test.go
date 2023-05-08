package amm_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpamm "github.com/NibiruChain/nibiru/x/perp/amm"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestSnapshotUpdates(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	perpammKeeper := nibiruApp.PerpAmmKeeper

	runBlock := func(duration time.Duration) {
		perpamm.EndBlocker(ctx, nibiruApp.PerpAmmKeeper)
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
	}

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC)).WithBlockHeight(1)

	require.NoError(t, perpammKeeper.CreatePool(
		/* ctx */ ctx,
		/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		/* quoteReserve */ sdk.NewDec(1_000),
		/* baseReserve */ sdk.NewDec(1_000),
		/* config */ *types.DefaultMarketConfig().WithTradeLimitRatio(sdk.OneDec()).WithFluctuationLimitRatio(sdk.OneDec()),
		/* pegMultiplier */ sdk.OneDec(),
	))
	expectedSnapshot := types.NewReserveSnapshot(
		/* pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		/* baseReserve */ sdk.NewDec(1_000),
		/* quoteReserve */ sdk.NewDec(1_000),
		/* pegMultiplier */ sdk.OneDec(),
		/* blockTime */ ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err := perpammKeeper.ReserveSnapshots.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	t.Log("affect mark price")
	market, err := perpammKeeper.GetPool(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.NoError(t, err)
	_, baseAmtAbs, err := perpammKeeper.SwapQuoteForBase(
		ctx,
		market,
		types.Direction_LONG,
		sdk.NewDec(250), // ← dyAmm
		sdk.ZeroDec(),
		false,
	)
	// dxAmm := (k / (y + dyAmm)) - x = (1e6 / (1e3 + 250)) - 1e3 = -200
	assert.EqualValues(t, sdk.NewDec(200), baseAmtAbs)
	require.NoError(t, err)
	expectedSnapshot = types.NewReserveSnapshot(
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		sdk.NewDec(800),   // ← x + dxAmm
		sdk.NewDec(1_250), // ← y + dyAMM
		sdk.OneDec(),
		ctx.BlockTime(),
	)

	t.Log("run one block of 5 seconds")
	ctxAtSnapshot := sdk.Context(ctx) // manually copy ctx before the time skip
	timeSkipDuration := 5 * time.Second
	runBlock(timeSkipDuration) // increments ctx.blockHeight and ctx.BlockTime
	snapshot, err = perpammKeeper.ReserveSnapshots.Get(ctx,
		collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), time.UnixMilli(expectedSnapshot.TimestampMs)))
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	testutil.RequireContainsTypedEvent(t, ctx, &types.ReserveSnapshotSavedEvent{
		Pair:           expectedSnapshot.Pair,
		QuoteReserve:   expectedSnapshot.QuoteReserve,
		BaseReserve:    expectedSnapshot.BaseReserve,
		MarkPrice:      snapshot.QuoteReserve.Quo(snapshot.BaseReserve).Mul(snapshot.PegMultiplier),
		BlockHeight:    ctxAtSnapshot.BlockHeight(),
		BlockTimestamp: ctxAtSnapshot.BlockTime(),
	})
}
