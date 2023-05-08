package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type changeLiquidationFeeRatio struct {
	LiquidationFeeRatio sdk.Dec
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
