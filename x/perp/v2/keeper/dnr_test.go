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
