package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

type fundAccount struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func FundAccount(account sdk.AccAddress, amount sdk.Coins) Action {
	return &fundAccount{Account: account, Amount: amount}
}

func (c fundAccount) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, inflationtypes.ModuleName, c.Account, c.Amount)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

type fundModule struct {
	Module string
	Amount sdk.Coins
}

func FundModule(module string, amount sdk.Coins) Action {
	return fundModule{Module: module, Amount: amount}
}

func (c fundModule) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err
	}

	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, inflationtypes.ModuleName, c.Module, c.Amount)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}
