package action

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type closeMarket struct {
	pair asset.Pair
}

func (c closeMarket) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	app.PerpKeeperV2.CloseMarket(ctx, c.pair)
	return ctx, nil, true
}

func CloseMarket(pair asset.Pair) action.Action {
	return closeMarket{pair: pair}
}
