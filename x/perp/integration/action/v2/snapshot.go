package action

import (
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
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
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
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
