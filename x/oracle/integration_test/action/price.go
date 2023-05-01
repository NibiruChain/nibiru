package action

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func SetOraclePrice(pair asset.Pair, price sdk.Dec) action.Action {
	return &setPairPrice{
		Pair:  pair,
		Price: price,
	}
}

type setPairPrice struct {
	Pair  asset.Pair
	Price sdk.Dec
}

func (s setPairPrice) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.OracleKeeper.SetPrice(ctx, s.Pair, s.Price)

	return ctx, nil, true
}

func InsertPriceSnapshot(pair asset.Pair, time time.Time, price sdk.Dec) action.Action {
	return insertPriceSnapshot{
		Pair:  pair,
		Time:  time,
		Price: price,
	}
}

type insertPriceSnapshot struct {
	Pair  asset.Pair
	Time  time.Time
	Price sdk.Dec
}

func (s insertPriceSnapshot) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.OracleKeeper.PriceSnapshots.Insert(ctx, collections.Join(s.Pair, s.Time), types.PriceSnapshot{
		Pair:        s.Pair,
		Price:       s.Price,
		TimestampMs: s.Time.UnixMilli(),
	})

	return ctx, nil
}
