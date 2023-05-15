package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestAdmin_WithdrawFromInsuranceFund(t *testing.T) {
	expectBalance := func(
		want sdk.Int, t *testing.T, nibiru *app.NibiruApp, ctx sdk.Context,
	) {
		insuranceFund := nibiru.AccountKeeper.GetModuleAddress(v2types.PerpEFModuleAccount)
		balances := nibiru.BankKeeper.GetAllBalances(ctx, insuranceFund)
		got := balances.AmountOf(denoms.NUSD)
		require.EqualValues(t, want.String(), got.String())
	}

	setup := func() (nibiru *app.NibiruApp, ctx sdk.Context) {
		nibiru, ctx = testapp.NewNibiruTestAppAndContext(true)
		expectBalance(sdk.ZeroInt(), t, nibiru, ctx)
		return nibiru, ctx
	}

	fundModule := func(t *testing.T, amount sdk.Int, ctx sdk.Context, nibiru *app.NibiruApp) {
		coins := sdk.NewCoins(sdk.NewCoin(denoms.NUSD, amount))
		err := testapp.FundModuleAccount(
			nibiru.BankKeeper, ctx, v2types.PerpEFModuleAccount,
			coins,
		)
		require.NoError(t, err)
	}

	testCases := []testutil.FunctionTestCase{
		{
			Name: "withdraw all",
			Test: func() {
				nibiru, ctx := setup()
				admin := testutil.AccAddress()
				amountToFund := sdk.NewInt(420)
				fundModule(t, amountToFund, ctx, nibiru)

				amountToWithdraw := amountToFund
				err := nibiru.PerpKeeperV2.Admin().WithdrawFromInsuranceFund(
					ctx, amountToWithdraw, admin)
				require.NoError(t, err)

				require.EqualValues(t,
					amountToFund.String(),
					nibiru.BankKeeper.GetBalance(ctx, admin, denoms.NUSD).Amount.String(),
				)
			},
		},
		{
			Name: "withdraw too much - err",
			Test: func() {
				nibiru, ctx := setup()
				admin := testutil.AccAddress()
				amountToFund := sdk.NewInt(420)
				fundModule(t, amountToFund, ctx, nibiru)

				amountToWithdraw := amountToFund.MulRaw(5)
				err := nibiru.PerpKeeperV2.Admin().WithdrawFromInsuranceFund(
					ctx, amountToWithdraw, admin)
				require.Error(t, err)
			},
		},
	}

	testutil.RunFunctionTests(t, testCases)
}
