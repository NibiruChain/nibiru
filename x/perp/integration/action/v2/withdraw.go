package action

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type withdraw struct {
	Account sdk.AccAddress
	Amount  sdk.Int
	Pair    asset.Pair
}

func (w withdraw) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	market, err := app.PerpKeeperV2.Markets.Get(ctx, w.Pair)
	if err != nil {
		return ctx, err, true
	}

	err = app.PerpKeeperV2.Withdraw(ctx, market, w.Account, w.Amount)
	return ctx, err, true
}

func Withdraw(pair asset.Pair, account sdk.AccAddress, amount sdk.Int) action.Action {
	return withdraw{
		Account: account,
		Amount:  amount,
		Pair:    pair,
	}
}
