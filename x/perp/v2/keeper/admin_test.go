package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
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
		nibiru, ctx = testapp.NewNibiruTestAppAndContext()
		expectBalance(sdk.ZeroInt(), t, nibiru, ctx)
		return nibiru, ctx
	}

	fundModule := func(t *testing.T, amount sdkmath.Int, ctx sdk.Context, nibiru *app.NibiruApp) {
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
	amm := *mock.TestAMMDefault()
	app, ctx := testapp.NewNibiruTestAppAndContext()

	// Error because of invalid market
	market := types.DefaultMarket(pair).WithMaintenanceMarginRatio(sdk.NewDec(2))
	err := app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
		Market:          &market, // Invalid maintenance ratio
	})
	require.ErrorContains(t, err, "maintenance margin ratio ratio must be 0 <= ratio <= 1")

	// Error because of invalid amm
	err = app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: sdk.NewDec(-1),
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "init price multiplier must be > 0")

	// Set it correctly
	err = app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.NoError(t, err)

	lastVersion, err := app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), lastVersion.Version)

	// Check that amm and market have version 1
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), market.Version)

	// Fail since it already exists and it is not disabled
	err = app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "already exists")

	// Disable the market to test that we can create it again but with an increased version
	err = app.PerpKeeperV2.ChangeMarketEnabledParameter(ctx, pair, false)
	require.NoError(t, err)

	err = app.PerpKeeperV2.Admin().CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.NoError(t, err)

	lastVersion, err = app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), lastVersion.Version)

	// Check that amm and market have version 2
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), market.Version)
}
