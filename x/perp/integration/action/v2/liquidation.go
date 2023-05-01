package action

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type PairTraderTuple struct {
	Pair       asset.Pair
	Trader     sdk.AccAddress
	Successful bool
}

type multiLiquidate struct {
	pairTraderTuples []PairTraderTuple
	liquidator       sdk.AccAddress
	shouldAllFail    bool
}

func (m multiLiquidate) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	liquidationRequests := make([]*v2types.MsgMultiLiquidate_Liquidation, len(m.pairTraderTuples))

	for i, pairTraderTuple := range m.pairTraderTuples {
		liquidationRequests[i] = &v2types.MsgMultiLiquidate_Liquidation{
			Pair:   pairTraderTuple.Pair,
			Trader: pairTraderTuple.Trader.String(),
		}
	}

	responses, err := app.PerpKeeperV2.MultiLiquidate(ctx, m.liquidator, liquidationRequests)

	if m.shouldAllFail {
		// we check if all liquidations failed
		if err == nil {
			return ctx, fmt.Errorf("multi liquidations should have all failed, but instead some succeeded"), true
		}

		for i, response := range responses {
			if response.Success {
				return ctx, fmt.Errorf("multi liquidations should have all failed, but instead some succeeded, index %d", i), true
			}
		}

		return ctx, nil, true
	}

	// otherwise, some succeeded and some may have failed
	if err != nil {
		return ctx, err, true
	}

	for i, response := range responses {
		if response.Success != m.pairTraderTuples[i].Successful {
			return ctx, fmt.Errorf("MultiLiquidate wrong assertion, expected %v, got %v, index %d", m.pairTraderTuples[i].Successful, response.Success, i), false
		}
	}

	return ctx, nil, true
}

func MultiLiquidate(liquidator sdk.AccAddress, shouldAllFail bool, pairTraderTuples ...PairTraderTuple) action.Action {
	return multiLiquidate{
		pairTraderTuples: pairTraderTuples,
		liquidator:       liquidator,
		shouldAllFail:    shouldAllFail,
	}
}

type multiLiquidateAllShouldFail struct {
	pairTraderTuples []PairTraderTuple
	liquidator       sdk.AccAddress
}
