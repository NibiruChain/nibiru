package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
	"time"
)

func TestDisableMarket(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()
	alice := testutil.AccAddress()

	tc := TestCases{
		TC("market can be disabled").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(true),
				),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(false),
				),
			),
		TC("cannot open position on disabled market").
			Given(
				CreateCustomMarket(
					pairBtcUsdc,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketOrderFails(
					alice,
					pairBtcUsdc,
					types.Direction_LONG,
					sdk.NewInt(10_000),
					sdk.OneDec(),
					sdk.ZeroDec(),
					types.ErrMarketNotEnabled,
				),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
