package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestAdmin_WithdrawFromInsuranceFund(t *testing.T) {
	expectBalance := func(
		want sdkmath.Int, t *testing.T, nibiru *app.NibiruApp, ctx sdk.Context,
	) {
		insuranceFund := nibiru.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount)
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
			nibiru.BankKeeper, ctx, types.PerpEFModuleAccount,
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
				expectBalance(sdk.ZeroInt(), t, nibiru, ctx)
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

func TestEnableMarket(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("true -> false").
			Given(
				CreateCustomMarket(pair),
				MarketShouldBeEqual(pair, Market_EnableShouldBeEqualTo(true)),
			).
			When(
				SetMarketEnabled(pair, false),
				MarketShouldBeEqual(pair, Market_EnableShouldBeEqualTo(false)),
				SetMarketEnabled(pair, true),
			).
			Then(
				MarketShouldBeEqual(pair, Market_EnableShouldBeEqualTo(true)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestCreateMarket(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("create pool").
			Given().
			When(
				CreateMarket(pair, *mock.TestMarket(), *mock.TestAMMDefault()),
			).
			Then(
				MarketShouldBeEqual(pair,
					Market_EnableShouldBeEqualTo(true),
					Market_PrepaidBadDebtShouldBeEqualTo(sdk.ZeroInt()),
					Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec()),
				),
				AMMShouldBeEqual(pair,
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
				),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}
