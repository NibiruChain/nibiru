package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
	"time"
)

func TestDisableMarket(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := TestCases{
		TC("spot twap").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertReserveSnapshot(pairBtcUsdc, startTime, WithPriceMultiplier(sdk.NewDec(9))),
				InsertReserveSnapshot(pairBtcUsdc, startTime.Add(10*time.Second), WithPriceMultiplier(sdk.NewDec(10))),
				InsertReserveSnapshot(pairBtcUsdc, startTime.Add(20*time.Second), WithPriceMultiplier(sdk.NewDec(11))),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Second),
			).
			Then(),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
