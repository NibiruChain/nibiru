package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	tokenfactorytypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

type fundAccount struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func FundAccount(account sdk.AccAddress, amount sdk.Coins) Action {
	return &fundAccount{Account: account, Amount: amount}
}

func (c fundAccount) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, c.Account, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}

type fundModule struct {
	Module string
	Amount sdk.Coins
}

func FundModule(module string, amount sdk.Coins) Action {
	return fundModule{Module: module, Amount: amount}
}

func (c fundModule) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, tokenfactorytypes.ModuleName, c.Module, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}
