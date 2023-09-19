package perp_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	perp "github.com/NibiruChain/nibiru/x/perp/v2/module"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestSnapshotUpdates(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()
	initialMarket := *mock.TestMarket()
	initialAmm := *mock.TestAMMDefault()

	runBlock := func(duration time.Duration) {
		perp.EndBlocker(ctx, app.PerpKeeperV2)
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
	}

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC)).WithBlockHeight(1)

	require.NoError(t, app.PerpKeeperV2.Admin().CreateMarket(
		/* ctx */ ctx, keeper.ArgsCreateMarket{
			Pair:            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			PriceMultiplier: initialAmm.PriceMultiplier,
			SqrtDepth:       initialAmm.SqrtDepth,
			Market:          &initialMarket,
		},
	))

	expectedSnapshot := types.ReserveSnapshot{
		Amm:         initialAmm,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	}

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)
	snapshot, err := app.PerpKeeperV2.ReserveSnapshots.Get(ctx,
		collections.Join(
			asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			time.UnixMilli(expectedSnapshot.TimestampMs),
		),
	)
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)

	t.Log("affect mark price")
	market, err := app.PerpKeeperV2.GetMarket(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.NoError(t, err)
	amm, err := app.PerpKeeperV2.GetAMM(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD))
	require.NoError(t, err)

	_, baseAmtAbs, err := app.PerpKeeperV2.SwapQuoteAsset(
		ctx,
		market,
		amm,
		types.Direction_LONG,
		sdk.NewDec(250e9),
		sdk.ZeroDec(),
	)
	require.NoError(t, err)
	// dxAmm := (k / (y + dyAmm)) - x = (1e24 / (1e12 + 250e9)) - 1e12 = -200e9
	assert.EqualValues(t, sdk.NewDec(200e9), baseAmtAbs)

	expectedSnapshot = types.ReserveSnapshot{
		Amm: types.AMM{
			Pair:            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			Version:         1,
			BaseReserve:     sdk.NewDec(800e9),
			QuoteReserve:    sdk.NewDec(1.25e12),
			SqrtDepth:       sdk.NewDec(1e12),
			PriceMultiplier: sdk.OneDec(),
			TotalLong:       sdk.NewDec(200e9),
			TotalShort:      sdk.ZeroDec(),
		},
		TimestampMs: ctx.BlockTime().UnixMilli(),
	}

	t.Log("run one block of 5 seconds")
	runBlock(5 * time.Second)

	snapshot, err = app.PerpKeeperV2.ReserveSnapshots.Get(ctx,
		collections.Join(
			asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			time.UnixMilli(expectedSnapshot.TimestampMs),
		),
	)
	require.NoError(t, err)
	assert.EqualValues(t, expectedSnapshot, snapshot)
}

func TestEndBlocker(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	initialMarket := *mock.TestMarket()
	initialAmm := *mock.TestAMMDefault()

	runBlock := func(duration time.Duration) {
		perp.EndBlocker(ctx, app.PerpKeeperV2)
		ctx = ctx.
			WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(duration))
	}

	ctx = ctx.WithBlockTime(time.Date(2015, 10, 21, 0, 0, 0, 0, time.UTC)).WithBlockHeight(1)

	runBlock(5 * time.Second)

	require.NoError(t, app.PerpKeeperV2.Admin().CreateMarket(
		/* ctx */ ctx, keeper.ArgsCreateMarket{
			Pair:            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			PriceMultiplier: initialAmm.PriceMultiplier,
			SqrtDepth:       initialAmm.SqrtDepth,
			Market:          &initialMarket,
		},
	))

	t.Log("run one block of 5 seconds")
	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.NewDec(100e6))

	beforeEvents := ctx.EventManager().Events()
	runBlock(5 * time.Second)
	afterEvents := ctx.EventManager().Events()

	testutilevents.AssertEventsPresent(
		t,
		testutilevents.FilterNewEvents(beforeEvents, afterEvents),
		[]string{"nibiru.perp.v2.AmmUpdatedEvent", "nibiru.perp.v2.MarketUpdatedEvent"},
	)

	// add index price
}
