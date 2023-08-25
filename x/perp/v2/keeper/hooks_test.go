package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	epochsaction "github.com/NibiruChain/nibiru/x/epochs/integration/action"
	epochtypes "github.com/NibiruChain/nibiru/x/epochs/types"
	oracleaction "github.com/NibiruChain/nibiru/x/oracle/integration/action"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassert "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
)

func TestAfterEpochEnd(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := tutilaction.TestCases{
		tutilaction.TC("index > mark").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.SetBlockTime(startTime),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("-0.099999999999999999"))),
			),

		tutilaction.TC("index < mark").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.SetBlockTime(startTime),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("0.01"))),
			),

		tutilaction.TC("index > mark - max funding rate").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc, perpaction.WithMaxFundingRate(sdk.MustNewDecFromStr("0.001"))),
				tutilaction.SetBlockTime(startTime),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("-0.000120833333333333"))),
			),

		tutilaction.TC("index < mark - max funding rate").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc, perpaction.WithMaxFundingRate(sdk.MustNewDecFromStr("0.001"))),
				tutilaction.SetBlockTime(startTime),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("0.000010833333333333"))),
			),

		tutilaction.TC("index == mark").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.SetBlockTime(startTime),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.OneDec()),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		tutilaction.TC("missing twap").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.SetBlockTime(startTime),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		tutilaction.TC("0 price mark").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				tutilaction.SetBlockTime(startTime),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		tutilaction.TC("market not enabled").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.SetMarketEnabled(pairBtcUsdc, false),
				tutilaction.SetBlockTime(startTime),
				epochsaction.StartEpoch(epochtypes.ThirtyMinuteEpochID),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.NewDec(2)),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		tutilaction.TC("not correct epoch id").
			Given(
				perpaction.CreateCustomMarket(pairBtcUsdc),
				perpaction.SetMarketEnabled(pairBtcUsdc, false),
				tutilaction.SetBlockTime(startTime),
				epochsaction.StartEpoch(epochtypes.FifteenMinuteEpochID),
				oracleaction.InsertOraclePriceSnapshot(pairBtcUsdc, startTime.Add(15*time.Minute), sdk.NewDec(2)),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				perpassert.MarketShouldBeEqual(pairBtcUsdc, perpassert.Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}
