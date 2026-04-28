package assertion

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
)

func AllBalancesEqual(
	nibiru *app.NibiruApp, ctx sdk.Context, account sdk.AccAddress, amount sdk.Coins,
) error {
	coins := nibiru.BankKeeper.GetAllBalances(ctx, account)
	if !coins.IsEqual(amount) {
		return fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			account.String(),
			amount.String(),
			coins.String(),
		)
	}

	return nil
}

func BalanceEqual(
	nibiru *app.NibiruApp, ctx sdk.Context,
	account sdk.AccAddress, denom string, amount sdkmath.Int,
) error {
	coin := nibiru.BankKeeper.GetBalance(ctx, account, denom)
	if !coin.Amount.Equal(amount) {
		return fmt.Errorf(
			"account %s balance not equal, expected %s, got %s",
			account.String(),
			amount.String(),
			coin.String(),
		)
	}

	return nil
}

func ModuleBalanceEqual(
	nibiru *app.NibiruApp, ctx sdk.Context,
	moduleName string, denom string, amount sdkmath.Int,
) error {
	coin := nibiru.BankKeeper.GetBalance(
		ctx,
		nibiru.AccountKeeper.GetModuleAddress(moduleName),
		denom,
	)
	if !coin.Amount.Equal(amount) {
		return fmt.Errorf(
			"module %s balance not equal, expected %s, got %s",
			moduleName,
			amount.String(),
			coin.String(),
		)
	}

	return nil
}
