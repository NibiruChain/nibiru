package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type liquidatePosition struct {
	Liquidator sdk.AccAddress
	Account    sdk.AccAddress
	Pair       asset.Pair
}

func (l liquidatePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, _, err := app.PerpKeeperV2.Liquidate(ctx, l.Liquidator, l.Pair, l.Account)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}

func LiquidatePosition(liquidator, account sdk.AccAddress, pair asset.Pair) action.Action {
	return liquidatePosition{
		Liquidator: liquidator,
		Account:    account,
		Pair:       pair,
	}
}

type setLiquidator struct {
	Account sdk.AccAddress
}

func (s setLiquidator) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	// TODO(k-yang): set liquidator on v2types.Market object

	return ctx, nil, true
}

func SetLiquidator(account sdk.AccAddress) action.Action {
	return setLiquidator{
		Account: account,
	}
}

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
