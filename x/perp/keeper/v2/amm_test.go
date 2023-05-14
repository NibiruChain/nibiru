package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestEditPriceMultipler(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("same price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.OneDec()),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("net bias zero").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("long bias, increase price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1004500)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(995500)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("long bias, decrease price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(999626)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1000374)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),

		TC("short bias, increase price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(500)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(995500)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1004500)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("short bias, decrease price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(500)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1000376)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(999624)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestEditSwapInvariant(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("same swap invariant").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)),
					WithTotalShort(sdk.NewDec(500)),
					WithSqrtDepth(sdk.NewDec(1e6)),
				),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e12)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e6)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e6)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e6)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("net bias zero").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)),
					WithTotalShort(sdk.NewDec(1000)),
					WithSqrtDepth(sdk.NewDec(1e6)),
				),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e18)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e9)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e9)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e9)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("long bias, increase swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1e5)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1008101)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(991899)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("long bias, decrease swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1e2)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(999991)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1000009)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("short bias, increase swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalShort(sdk.NewDec(1e5)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1010102)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(989898)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("short bias, decrease swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalShort(sdk.NewDec(1e2)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				ModuleBalanceShouldBeEqual(v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(999989)))),
				ModuleBalanceShouldBeEqual(v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1000011)))),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}
