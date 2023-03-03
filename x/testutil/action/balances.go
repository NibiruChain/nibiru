package action

import (
	"fmt"

	"github.com/NibiruChain/nibiru/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BalanceEqual(account sdk.AccAddress, amount sdk.Coins) *BalanceEqualAction {
	return &BalanceEqualAction{Account: account, Amount: amount}
}

type BalanceEqualAction struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func (b BalanceEqualAction) Do(app *app.NibiruApp, ctx sdk.Context) error {
	acc := app.AccountKeeper.GetAccount(ctx, b.Account)
	if acc == nil {
		return fmt.Errorf("account %s not found", b.Account.String())
	}

	coins := app.BankKeeper.GetAllBalances(ctx, b.Account)
	if !coins.IsEqual(b.Amount) {
		return fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			b.Account.String(),
			b.Amount.String(),
			coins.String(),
		)
	}

	return nil
}
