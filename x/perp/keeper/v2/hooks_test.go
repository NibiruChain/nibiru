package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/epochs/integration/action"
	epochtypes "github.com/NibiruChain/nibiru/x/epochs/types"
	. "github.com/NibiruChain/nibiru/x/oracle/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
)

func TestAfterEpochEnd(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := TestCases{
		TC("index > mark").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("-0.1"))),
			),

		TC("index < mark").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("0.01"))),
			),

		TC("index == mark").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.OneDec()),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
