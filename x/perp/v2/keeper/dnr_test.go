package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestUserVolumes(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	positionSize := sdk.NewInt(10_000)
	startBlockTime := time.Now()

	tests := TestCases{
		TC("user volume correctly sets the first time and the second time").
			Given(
				DnREpochIs(1),
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			Then(
				DnRCurrentVolumeIs(alice, positionSize.MulRaw(2)),
			),
		TC("user volume correctly sets across epochs").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				DnREpochIs(1),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()), // open epoch 1
				DnREpochIs(2),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()), // close epoch 2
			).
			Then(
				DnRCurrentVolumeIs(alice, positionSize),
				DnRPreviousVolumeIs(alice, positionSize),
			),
		TC("user volume is correctly cleaned up").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).When(
			DnREpochIs(1),
			MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()), // open epoch 1
			DnREpochIs(2),
			MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(5_000), sdk.OneDec(), sdk.ZeroDec()), // reduce epoch 2
			DnREpochIs(3),
			MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(2_000), sdk.OneDec(), sdk.ZeroDec()), // reduce epoch 3
			SetBlockNumber(keeper.DnRGCFrequency),
			MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(2_000), sdk.OneDec(), sdk.ZeroDec()), // reduce more epoch 3
		).
			Then(
				DnRCurrentVolumeIs(alice, math.NewInt(4000)),  // for current epoch only 4k in volume.
				DnRPreviousVolumeIs(alice, math.NewInt(5000)), // for previous epoch only 5k in volume.
				DnRVolumeNotExist(alice, 1),                   // volume for epoch 1 should not exist.
			),
	}
	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestDiscountAndRebates(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	positionSize := sdk.NewInt(10_000)
	startBlockTime := time.Now()

	exchangeFee := sdk.MustNewDecFromStr("0.0010")           // 0.1%, default fee taken from CreateCustomMarketAction
	globalFeeDiscount := sdk.MustNewDecFromStr("0.0005")     // 0.05%
	fauxGlobalFeeDiscount := sdk.MustNewDecFromStr("0.0006") // 0.06%
	customFeeDiscount := sdk.MustNewDecFromStr("0.0002")     // 0.02%
	fauxCustomFeeDiscount := sdk.MustNewDecFromStr("0.0003") // 0.03%
	customRebate := sdk.MustNewDecFromStr("0.001")           // 0.01%

	tests := TestCases{
		TC("user does not have any past epoch volume: no discount applies").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				DnREpochIs(1),
			).
			Then(
				MarketOrderFeeAndRebateIs(exchangeFee, math.ZeroInt(), alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			),
		TC("user has past epoch volume: no discount applies").
			Given(
				DnREpochIs(2),
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				SetGlobalDiscount(fauxGlobalFeeDiscount, sdk.NewInt(50_000)),
				SetGlobalDiscount(globalFeeDiscount, sdk.NewInt(100_000)),
				SetCustomDiscount(alice, fauxCustomFeeDiscount, sdk.NewInt(50_000)),
				SetCustomDiscount(alice, customFeeDiscount, sdk.NewInt(100_000)),
				SetPreviousEpochUserVolume(alice, sdk.NewInt(10_000)), // lower than 50_000
			).
			Then(
				MarketOrderFeeAndRebateIs(exchangeFee, math.ZeroInt(), alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			),
		TC("user has past epoch volume: custom discount applies").
			Given(
				DnREpochIs(2),
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				SetGlobalDiscount(globalFeeDiscount, sdk.NewInt(50_000)),
				SetGlobalDiscount(fauxGlobalFeeDiscount, sdk.NewInt(100_000)),
				SetCustomDiscount(alice, fauxCustomFeeDiscount, sdk.NewInt(50_000)),
				SetCustomDiscount(alice, customFeeDiscount, sdk.NewInt(100_000)),
				SetPreviousEpochUserVolume(alice, sdk.NewInt(100_001)),
			).
			Then(
				MarketOrderFeeAndRebateIs(customFeeDiscount, math.ZeroInt(), alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			),

		TC("trades do not fail if coinToBondEquivalent function fails").
			Given(
				DnREpochIs(2),
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
				// we do not set the price here!
			).
			When(
				SetCustomRebate(alice, customRebate, sdk.NewInt(99_999)), // alice qualifies for the rebate but the asset is unknown
				SetPreviousEpochUserVolume(alice, sdk.NewInt(100_000)),
			).
			Then(
				MarketOrderFeeAndRebateIs(exchangeFee, math.ZeroInt(), alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			),
		TC("if bond denom equals to quote denom then equivalent is minted").
			Given(
				DnREpochIs(2),
				CreateCustomMarket(
					asset.NewPair("btc", "stake"),
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin("stake", positionSize.AddRaw(1000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100_000_000)))),
				// we do not set the price here, because if quote denom is bond denom, then we should just return 1 as price.
			).
			When(
				SetCustomRebate(alice, customRebate, sdk.NewInt(99_999)), // alice qualifies for the rebate but the asset is unknown
				SetPreviousEpochUserVolume(alice, sdk.NewInt(100_000)),
			).
			Then(
				// customRebate does not need adjustments with respect to price since quote and bond denom are the same.
				MarketOrderFeeAndRebateIs(exchangeFee, customRebate.MulInt(sdk.NewInt(10_000)).TruncateInt(), alice, asset.NewPair("btc", "stake"), types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
			),
	}
	NewTestSuite(t).WithTestCases(tests...).Run()
}
