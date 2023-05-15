package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type changeLiquidationFeeRatio struct {
	LiquidationFeeRatio sdk.Dec
}

type setMarketEnabled struct {
	Enable bool
	Pair   asset.Pair
}

func (c changeLiquidationFeeRatio) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	// TODO(k-yang): implement

	return ctx, nil, true
}

func ChangeLiquidationFeeRatio(liquidationFeeRatio sdk.Dec) action.Action {
	return changeLiquidationFeeRatio{
		LiquidationFeeRatio: liquidationFeeRatio,
	}
}

// Enable market
func SetMarketEnabled(pair asset.Pair, enable bool) action.Action {
	return setMarketEnabled{
		Enable: enable,
		Pair:   pair,
	}
}

func (c setMarketEnabled) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.Admin().SetMarketEnabled(ctx, c.Pair, c.Enable)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}
