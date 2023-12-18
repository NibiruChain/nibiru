package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type moduleBalanceShouldBeEqual struct {
	module          string
	expectedBalance sdk.Coins
}

func (p moduleBalanceShouldBeEqual) IsNotMandatory() {}

func (p moduleBalanceShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	balance := app.BankKeeper.GetAllBalances(ctx, app.AccountKeeper.GetModuleAddress(p.module))

	if !balance.IsEqual(p.expectedBalance) {
		return ctx, fmt.Errorf("balance expected for %s to be %s, received %s", p.module, p.expectedBalance.String(), balance.String())
	}

	return ctx, nil
}

func ModuleBalanceShouldBeEqualTo(module string, expectedBalance sdk.Coins) action.Action {
	return moduleBalanceShouldBeEqual{
		module:          module,
		expectedBalance: expectedBalance,
	}
}
