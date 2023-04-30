package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type changeMaintenanceMarginRatio struct {
	MaintenanceMarginRatio sdk.Dec
	Pair                   asset.Pair
}

func (c changeMaintenanceMarginRatio) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	pool, err := app.PerpAmmKeeper.GetPool(ctx, c.Pair)
	if err != nil {
		return ctx, err, true
	}

	pool.Config.MaintenanceMarginRatio = c.MaintenanceMarginRatio
	app.PerpAmmKeeper.Pools.Insert(ctx, c.Pair, pool)
	return ctx, err, true
}

func ChangeMaintenanceMarginRatio(pair asset.Pair, maintenanceMarginRatio sdk.Dec) action.Action {
	return changeMaintenanceMarginRatio{
		Pair:                   pair,
		MaintenanceMarginRatio: maintenanceMarginRatio,
	}
}
