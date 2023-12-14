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
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
)

func TestAfterEpochEnd(t *testing.T) {
	pairBtcUsd := asset.Registry.Pair(denoms.BTC, denoms.USD)
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := TestCases{
		TC("index > mark").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("-0.099999999999999999"))),
			),

		TC("index < mark").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("0.01"))),
			),

		TC("index > mark - max funding rate").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true), WithMaxFundingRate(sdk.MustNewDecFromStr("0.001"))),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("5.8")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("-0.000120833333333333"))),
			),

		TC("index < mark - max funding rate").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true), WithMaxFundingRate(sdk.MustNewDecFromStr("0.001"))),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.MustNewDecFromStr("0.52")),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.MustNewDecFromStr("0.000010833333333333"))),
			),

		TC("index == mark").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.OneDec()),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		TC("missing twap").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		TC("0 price mark").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		TC("market closed").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				CloseMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				StartEpoch(epochtypes.ThirtyMinuteEpochID),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.NewDec(2)),
			).
			When(
				MoveToNextBlockWithDuration(30 * time.Minute),
			).
			Then(
				MarketShouldBeEqual(pairBtcUsdc, Market_LatestCPFShouldBeEqualTo(sdk.ZeroDec())),
			),

		TC("not correct epoch id").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				CloseMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				StartEpoch(epochtypes.DayEpochID),
				InsertOraclePriceSnapshot(pairBtcUsd, startTime.Add(15*time.Minute), sdk.NewDec(2)),
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
