package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	tutilassert "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassert "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestWithdrawFromVault(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := tutilaction.TestCases{
		tutilaction.TC("successful withdraw, no bad debt").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1000)))),
			).
			When(
				perpaction.WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				tutilassert.BalanceEqual(alice, denoms.USDC, sdk.NewInt(1000)),
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.ZeroInt()),
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_PrepaidBadDebtShouldBeEqualTo(sdk.ZeroInt())),
			),

		tutilaction.TC("successful withdraw, some bad debt").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(500)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(500)))),
			).
			When(
				perpaction.WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				tutilassert.BalanceEqual(alice, denoms.USDC, sdk.NewInt(1000)),
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.ZeroInt()),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(500))),
			),

		tutilaction.TC("successful withdraw, all bad debt").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1000)))),
			).
			When(
				perpaction.WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				tutilassert.BalanceEqual(alice, denoms.USDC, sdk.NewInt(1000)),
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.ZeroInt()),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(1000))),
			),

		tutilaction.TC("successful withdraw, existing bad debt").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.CreateCustomMarket(pairBtcUsdc, perpaction.WithPrepaidBadDebt(sdk.NewInt(1000))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1000)))),
			).
			When(
				perpaction.WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				tutilassert.BalanceEqual(alice, denoms.USDC, sdk.NewInt(1000)),
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.ZeroInt()),
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.ZeroInt()),
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(2000))),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}
