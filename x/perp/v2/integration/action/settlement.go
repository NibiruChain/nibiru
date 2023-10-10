package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type closeMarket struct {
	pair asset.Pair
}

func (c closeMarket) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.CloseMarket(ctx, c.pair)
	if err != nil {
		return ctx, err, false
	}

	return ctx, nil, true
}

func CloseMarket(pair asset.Pair) action.Action {
	return closeMarket{pair: pair}
}
