package action

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type liquidatePosition struct {
	Liquidator sdk.AccAddress
	Account    sdk.AccAddress
	Pair       asset.Pair
}

func (l liquidatePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, _, err := app.PerpKeeper.Liquidate(ctx, l.Liquidator, l.Pair, l.Account)
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
	params := app.PerpKeeper.GetParams(ctx)

	params.WhitelistedLiquidators = append(params.WhitelistedLiquidators, s.Account.String())
	app.PerpKeeper.SetParams(ctx, params)

	return ctx, nil
}

func SetLiquidator(account sdk.AccAddress) action.Action {
	return setLiquidator{
		Account: account,
	}
}
