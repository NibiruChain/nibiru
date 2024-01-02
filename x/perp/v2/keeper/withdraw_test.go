package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestWithdrawFromVault(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("successful withdraw, no bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcUsdc),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
			).
			When(
				WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(1000)),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				MarketShouldBeEqual(pairBtcUsdc, Market_PrepaidBadDebtShouldBeEqualTo(sdk.ZeroInt(), types.TestingCollateralDenomNUSD)),
			),

		TC("successful withdraw, some bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcUsdc),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(500)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(500)))),
			).
			When(
				WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(1000)),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				MarketShouldBeEqual(pairBtcUsdc, Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(500), types.TestingCollateralDenomNUSD)),
			),

		TC("successful withdraw, all bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcUsdc),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
			).
			When(
				WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(1000)),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				MarketShouldBeEqual(pairBtcUsdc, Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(1000), types.TestingCollateralDenomNUSD)),
			),

		TC("successful withdraw, existing bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcUsdc, WithPrepaidBadDebt(sdk.NewInt(1000), types.TestingCollateralDenomNUSD)),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
			).
			When(
				WithdrawFromVault(pairBtcUsdc, alice, sdk.NewInt(1000)),
			).
			Then(
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(1000)),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				MarketShouldBeEqual(pairBtcUsdc, Market_PrepaidBadDebtShouldBeEqualTo(sdk.NewInt(2000), types.TestingCollateralDenomNUSD)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
