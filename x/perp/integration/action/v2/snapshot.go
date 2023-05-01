package action

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type insertReserveSnapshot struct {
	pair asset.Pair
	time time.Time

	modifiers []reserveSnapshotModifier
}

func (i insertReserveSnapshot) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	amm := app.PerpKeeperV2.AMMs.GetOr(ctx, i.pair, v2types.AMM{
		Pair:            i.pair,
		BaseReserve:     sdk.ZeroDec(),
		QuoteReserve:    sdk.ZeroDec(),
		SqrtDepth:       sdk.ZeroDec(),
		PriceMultiplier: sdk.ZeroDec(),
		Bias:            sdk.ZeroDec(),
	})

	for _, modifier := range i.modifiers {
		modifier(&amm)
	}

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(i.pair, i.time), v2types.ReserveSnapshot{
		TimestampMs: i.time.UnixMilli(),
		Amm:         amm,
	})

	return ctx, nil, true
}

func InsertReserveSnapshot(pair asset.Pair, time time.Time, modifiers ...reserveSnapshotModifier) action.Action {
	return insertReserveSnapshot{
		pair:      pair,
		time:      time,
		modifiers: modifiers,
	}
}

type reserveSnapshotModifier func(amm *v2types.AMM)

func WithPriceMultiplier(multiplier sdk.Dec) reserveSnapshotModifier {
	return func(amm *v2types.AMM) {
		amm.PriceMultiplier = multiplier
	}
}

type calcTwap struct {
	pair               asset.Pair
	twapCalcOpt        v2types.TwapCalcOption
	dir                v2types.Direction
	assetAmt           sdk.Dec
	twapLookbackWindow time.Duration

	expectedTwap sdk.Dec
}

func (c calcTwap) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	twap, err := app.PerpKeeperV2.CalcTwap(ctx, c.pair, c.twapCalcOpt, c.dir, c.assetAmt, c.twapLookbackWindow)
	if err != nil {
		return ctx, err, false
	}

	if !twap.Equal(c.expectedTwap) {
		return ctx, fmt.Errorf("invalid twap, expected %s, received %s", c.expectedTwap, twap), false
	}

	return ctx, nil, false
}

func CalcTwap(pair asset.Pair, twapCalcOpt v2types.TwapCalcOption, dir v2types.Direction, assetAmt sdk.Dec, twapLookbackWindow time.Duration, expectedTwap sdk.Dec) action.Action {
	return calcTwap{
		pair:               pair,
		twapCalcOpt:        twapCalcOpt,
		dir:                dir,
		assetAmt:           assetAmt,
		twapLookbackWindow: twapLookbackWindow,
		expectedTwap:       expectedTwap,
	}
}
