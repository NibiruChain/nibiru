package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

// AddMargin adds margin to the position
func AddMargin(
	account sdk.AccAddress,
	pair asset.Pair,
	margin sdk.Int,
) action.Action {
	return &addMarginAction{
		Account: account,
		Pair:    pair,
		Margin:  margin,
	}
}

type addMarginAction struct {
	Account sdk.AccAddress
	Pair    asset.Pair
	Margin  sdk.Int
}

func (a addMarginAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.AddMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(a.Pair.QuoteDenom(), a.Margin),
	)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}
