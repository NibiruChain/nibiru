package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type PairTraderTuple struct {
	Pair   asset.Pair
	Trader sdk.AccAddress
}

type multiLiquidate struct {
	pairTraderTuples []PairTraderTuple
	liquidator       sdk.AccAddress
}

func (m multiLiquidate) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	liquidationRequests := make([]*v2types.MsgMultiLiquidate_Liquidation, 0)

	for _, pairTraderTuple := range m.pairTraderTuples {
		liquidationRequests = append(liquidationRequests, &v2types.MsgMultiLiquidate_Liquidation{
			Pair:   pairTraderTuple.Pair,
			Trader: pairTraderTuple.Trader.String(),
		})
	}

	_, err := app.PerpKeeperV2.MultiLiquidate(ctx, m.liquidator, liquidationRequests)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}

func MultiLiquidate(liquidator sdk.AccAddress, pairTraderTuples ...PairTraderTuple) action.Action {
	return multiLiquidate{
		pairTraderTuples: pairTraderTuples,
		liquidator:       liquidator,
	}
}
