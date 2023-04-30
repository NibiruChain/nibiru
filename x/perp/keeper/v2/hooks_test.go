package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"

	// . "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"

	// . "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
)

func TestAfterEpochEnd(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := TestCases{
		TC("index > mark").
			Given(
				createInitMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertPriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketLatestCPFShouldBeEqual(pairBtcUsdc, sdk.MustNewDecFromStr("-0.1")),
			),

		TC("index < mark").
			Given(
				createInitMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertPriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketLatestCPFShouldBeEqual(pairBtcUsdc, sdk.MustNewDecFromStr("0.01")),
			),

		TC("index == mark").
			Given(
				createInitMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertPriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.OneDec()),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketLatestCPFShouldBeEqual(pairBtcUsdc, sdk.ZeroDec()),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
