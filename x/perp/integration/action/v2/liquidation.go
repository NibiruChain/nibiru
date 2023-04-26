package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type liquidatePosition struct {
	Liquidator sdk.AccAddress
	Account    sdk.AccAddress
	Pair       asset.Pair
}

func (l liquidatePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, _, err := app.PerpKeeperV2.Liquidate(ctx, l.Liquidator, l.Pair, l.Account)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
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

func (s setLiquidator) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	// TODO(k-yang): set liquidator on v2types.Market object

	return ctx, nil
}

func SetLiquidator(account sdk.AccAddress) action.Action {
	return setLiquidator{
		Account: account,
	}
}
