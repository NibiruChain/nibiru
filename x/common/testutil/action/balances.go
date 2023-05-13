package action

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
)

func BalanceShouldBeEqual(account sdk.AccAddress, amount sdk.Coins) *balanceShouldBeEqual {
	return &balanceShouldBeEqual{Account: account, Amount: amount}
}

type balanceShouldBeEqual struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func (b balanceShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	acc := app.AccountKeeper.GetAccount(ctx, b.Account)
	if acc == nil {
		return ctx, fmt.Errorf("account %s not found", b.Account.String()), true
	}

	coins := app.BankKeeper.GetAllBalances(ctx, b.Account)
	if !coins.IsEqual(b.Amount) {
		return ctx, fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			b.Account.String(),
			b.Amount.String(),
			coins.String(),
		), true
	}

	return ctx, nil, true
}

type moduleBalanceShouldBeEqual struct {
	Module string
	Amount sdk.Coins
}

func (b moduleBalanceShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	moduleAddr := app.AccountKeeper.GetModuleAddress(b.Module)

	coins := app.BankKeeper.GetAllBalances(ctx, moduleAddr)
	if !coins.IsEqual(b.Amount) {
		return ctx, fmt.Errorf(
			"module %s balance not equal, expected %s, got %s",
			b.Module,
			b.Amount.String(),
			coins.String(),
		), true
	}

	return ctx, nil, true
}

func ModuleBalanceShouldBeEqual(module string, amount sdk.Coins) *moduleBalanceShouldBeEqual {
	return &moduleBalanceShouldBeEqual{Module: module, Amount: amount}
}
