package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

type fundAccount struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func FundAccount(account sdk.AccAddress, amount sdk.Coins) Action {
	return &fundAccount{Account: account, Amount: amount}
}

func (c fundAccount) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, inflationtypes.ModuleName, c.Account, c.Amount)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}
