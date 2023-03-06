package action

import (
	"github.com/NibiruChain/nibiru/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/NibiruChain/nibiru/app"
)

type fundAccount struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func FundAccount(account sdk.AccAddress, amount sdk.Coins) testutil.Action {
	return &fundAccount{Account: account, Amount: amount}
}

func (c fundAccount) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, c.Account, c.Amount)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}
