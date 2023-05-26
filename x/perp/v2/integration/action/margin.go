package action

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

// AddMargin adds margin to the position
func AddMargin(
	account sdk.AccAddress,
	pair asset.Pair,
	margin sdkmath.Int,
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
	Margin  sdkmath.Int
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

func RemoveMargin(
	account sdk.AccAddress,
	pair asset.Pair,
	margin sdkmath.Int,
) action.Action {
	return &removeMarginAction{
		Account: account,
		Pair:    pair,
		Margin:  margin,
	}
}

type removeMarginAction struct {
	Account sdk.AccAddress
	Pair    asset.Pair
	Margin  sdkmath.Int
}

func (a removeMarginAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.RemoveMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(a.Pair.QuoteDenom(), a.Margin),
	)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}
