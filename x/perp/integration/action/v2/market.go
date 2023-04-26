package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/testutil/action"

	"github.com/NibiruChain/nibiru/app"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// CreateMarketAction creates a market
type CreateMarketAction struct {
	Market v2types.Market
}

func (c CreateMarketAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.PerpKeeperV2.Markets.Insert(ctx, c.Market.Pair, c.Market)

	return ctx, nil
}

// CreateCustomMarket creates a market with custom parameters
func CreateCustomMarket(
	market v2types.Market,
) action.Action {
	return CreateMarketAction{
		Market: market,
	}
}
