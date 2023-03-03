package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/NibiruChain/nibiru/app"
)

type FundAccountAction struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func FundAccount(account sdk.AccAddress, amount sdk.Coins) *FundAccountAction {
	return &FundAccountAction{Account: account, Amount: amount}
}

func (c FundAccountAction) Do(app *app.NibiruApp, ctx sdk.Context) error {
	err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, c.Amount)
	if err != nil {
		return err
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, c.Account, c.Amount)
	if err != nil {
		return err
	}
	return nil
}
