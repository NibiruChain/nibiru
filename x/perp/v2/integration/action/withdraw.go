package action

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type withdraw struct {
	Account sdk.AccAddress
	Amount  sdkmath.Int
	Pair    asset.Pair
}

func (w withdraw) Do(app *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	market, err := app.PerpKeeperV2.GetMarket(ctx, w.Pair)
	if err != nil {
		return ctx, err, true
	}

	err = app.PerpKeeperV2.WithdrawFromVault(ctx, market, w.Account, w.Amount)
	return ctx, err, true
}

func WithdrawFromVault(pair asset.Pair, account sdk.AccAddress, amount sdkmath.Int) action.Action {
	return withdraw{
		Account: account,
		Amount:  amount,
		Pair:    pair,
	}
}
