package assertion

import (
	"fmt"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
)

func AllBalancesEqual(account sdk.AccAddress, amount sdk.Coins) action.Action {
	return &allBalancesEqual{Account: account, Amount: amount}
}

type allBalancesEqual struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func (b allBalancesEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	coins := app.BankKeeper.GetAllBalances(ctx, b.Account)
	if !coins.IsEqual(b.Amount) {
		return ctx, fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			b.Account.String(),
			b.Amount.String(),
			coins.String(),
		)
	}

	return ctx, nil
}

func BalanceEqual(account sdk.AccAddress, denom string, amount sdkmath.Int) action.Action {
	return &balanceEqual{Account: account, Denom: denom, Amount: amount}
}

type balanceEqual struct {
	Account sdk.AccAddress
	Denom   string
	Amount  sdkmath.Int
}

func (b balanceEqual) IsNotMandatory() {}

func (b balanceEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	coin := app.BankKeeper.GetBalance(ctx, b.Account, b.Denom)
	if !coin.Amount.Equal(b.Amount) {
		return ctx, fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			b.Account.String(),
			b.Amount.String(),
			coin.String(),
		)
	}

	return ctx, nil
}

func ModuleBalanceEqual(moduleName string, denom string, amount sdkmath.Int) action.Action {
	return &moduleBalanceEqual{ModuleName: moduleName, Denom: denom, Amount: amount}
}

type moduleBalanceEqual struct {
	ModuleName string
	Denom      string
	Amount     sdkmath.Int
}

func (b moduleBalanceEqual) IsNotMandatory() {}

func (b moduleBalanceEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	coin := app.BankKeeper.GetBalance(ctx, app.AccountKeeper.GetModuleAddress(b.ModuleName), b.Denom)
	if !coin.Amount.Equal(b.Amount) {
		return ctx, fmt.Errorf(
			"module %s balance not equal, expected %s, got %s",
			b.ModuleName,
			b.Amount.String(),
			coin.String(),
		)
	}

	return ctx, nil
}
