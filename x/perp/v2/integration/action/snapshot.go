package action

import (
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type insertReserveSnapshot struct {
	pair asset.Pair
	time time.Time

	modifiers []reserveSnapshotModifier
}

func (i insertReserveSnapshot) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	amm := app.PerpKeeperV2.AMMs.GetOr(ctx, collections.Join(i.pair, uint64(1)), types.AMM{
		Pair:            i.pair,
		Version:         uint64(1),
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

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(i.pair, i.time), types.ReserveSnapshot{
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

type reserveSnapshotModifier func(amm *types.AMM)

func WithPriceMultiplier(multiplier sdk.Dec) reserveSnapshotModifier {
	return func(amm *types.AMM) {
		amm.PriceMultiplier = multiplier
	}
}
