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

type settlePosition struct {
	pair    asset.Pair
	version uint64
	trader  sdk.AccAddress
}

func (c settlePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.SettlePosition(ctx, c.pair, c.version, c.trader)
	if err != nil {
		return ctx, err, false
	}

	return ctx, nil, true
}

func SettlePosition(pair asset.Pair, version uint64, trader sdk.AccAddress) action.Action {
	return settlePosition{pair: pair, version: version, trader: trader}
}

type settlePositionShouldFail struct {
	pair    asset.Pair
	version uint64
	trader  sdk.AccAddress
}

func (c settlePositionShouldFail) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.SettlePosition(ctx, c.pair, c.version, c.trader)
	if err == nil {
		return ctx, err, false
	}

	return ctx, nil, true
}

func SettlePositionShouldFail(pair asset.Pair, version uint64, trader sdk.AccAddress) action.Action {
	return settlePositionShouldFail{pair: pair, version: version, trader: trader}
}
