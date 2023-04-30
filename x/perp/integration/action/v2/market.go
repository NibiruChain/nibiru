package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"

	"github.com/NibiruChain/nibiru/app"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// CreateMarketAction creates a market
type CreateMarketAction struct {
	Market v2types.Market
	AMM    v2types.AMM
}

func (c CreateMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.PerpKeeperV2.Markets.Insert(ctx, c.Market.Pair, c.Market)
	app.PerpKeeperV2.AMMs.Insert(ctx, c.AMM.Pair, c.AMM)

	app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(c.AMM.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
		Amm:         c.AMM,
		TimestampMs: ctx.BlockTime().UnixMilli(),
	})

	return ctx, nil, true
}

// CreateCustomMarket creates a market with custom parameters
func CreateCustomMarket(
	market v2types.Market,
	amm v2types.AMM,
) action.Action {
	return CreateMarketAction{
		Market: market,
		AMM:    amm,
	}
}
