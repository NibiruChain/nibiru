package action

import (
	"errors"
	"fmt"

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
	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	_, err = app.PerpKeeperV2.AddMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(collateral.String(), a.Margin),
	)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}

// AddMarginFail adds margin to the position expecting a fail
func AddMarginFail(
	account sdk.AccAddress,
	pair asset.Pair,
	margin sdkmath.Int,
	err error,
) action.Action {
	return &addMarginFailAction{
		Account:     account,
		Pair:        pair,
		Margin:      margin,
		ExpectedErr: err,
	}
}

type addMarginFailAction struct {
	Account     sdk.AccAddress
	Pair        asset.Pair
	Margin      sdkmath.Int
	ExpectedErr error
}

func (a addMarginFailAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	_, err = app.PerpKeeperV2.AddMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(collateral.String(), a.Margin),
	)
	if !errors.Is(err, a.ExpectedErr) {
		return ctx, fmt.Errorf("expected error %v, got %v", a.ExpectedErr, err), false
	}

	return ctx, nil, false
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
	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	_, err = app.PerpKeeperV2.RemoveMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(collateral.String(), a.Margin),
	)
	if err != nil {
		return ctx, err, false
	}

	return ctx, nil, false
}

func RemoveMarginFail(
	account sdk.AccAddress,
	pair asset.Pair,
	margin sdkmath.Int,
	err error,
) action.Action {
	return &removeMarginActionFail{
		Account:     account,
		Pair:        pair,
		Margin:      margin,
		ExpectedErr: err,
	}
}

type removeMarginActionFail struct {
	Account     sdk.AccAddress
	Pair        asset.Pair
	Margin      sdkmath.Int
	ExpectedErr error
}

func (a removeMarginActionFail) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	_, err = app.PerpKeeperV2.RemoveMargin(
		ctx, a.Pair, a.Account, sdk.NewCoin(collateral.String(), a.Margin),
	)
	if !errors.Is(err, a.ExpectedErr) {
		return ctx, fmt.Errorf("expected error %v, got %v", a.ExpectedErr, err), false
	}

	return ctx, nil, false
}
