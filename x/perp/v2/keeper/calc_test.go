package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"

	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassert "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
)

func TestPositionNotionalSpot(t *testing.T) {
	tests := []struct {
		name             string
		amm              *types.AMM
		position         types.Position
		expectedNotional sdk.Dec
	}{
		{
			name: "long position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: types.Position{
				Size_: sdk.NewDec(10),
			},
			expectedNotional: sdk.MustNewDecFromStr("19.801980198019801980"),
		},
		{
			name: "short position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: types.Position{
				Size_: sdk.NewDec(-10),
			},
			expectedNotional: sdk.MustNewDecFromStr("20.202020202020202020"),
		},
		{
			name: "zero position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: types.Position{
				Size_: sdk.ZeroDec(),
			},
			expectedNotional: sdk.ZeroDec(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			notional, err := keeper.PositionNotionalSpot(*tc.amm, tc.position)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedNotional, notional)
		})
	}
}

func TestPositionNotionalTWAP(t *testing.T) {
	alice := testutil.AccAddress()
	pair := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startTime := time.Now()

	tc := tutilaction.TestCases{
		tutilaction.TC("long position").
			Given(
				tutilaction.SetBlockTime(startTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pair),
				perpaction.InsertPosition(perpaction.WithSize(sdk.NewDec(10)), perpaction.WithTrader(alice)),
				perpaction.InsertReserveSnapshot(pair, startTime, perpaction.WithPriceMultiplier(sdk.NewDec(9))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(10*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("8.5"))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(20*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("9.5"))),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Second),
			).
			Then(
				perpassert.PositionNotionalTWAPShouldBeEqualTo(pair, alice, 30*time.Second, sdk.MustNewDecFromStr("89.999999999100000000")),
			),

		tutilaction.TC("short position").
			Given(
				tutilaction.SetBlockTime(startTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pair),
				perpaction.InsertPosition(perpaction.WithSize(sdk.NewDec(-10)), perpaction.WithTrader(alice)),
				perpaction.InsertReserveSnapshot(pair, startTime, perpaction.WithPriceMultiplier(sdk.NewDec(9))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(10*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("8.5"))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(20*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("9.5"))),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Second),
			).
			Then(
				perpassert.PositionNotionalTWAPShouldBeEqualTo(pair, alice, 30*time.Second, sdk.MustNewDecFromStr("90.000000000900000000")),
			),

		tutilaction.TC("zero position").
			Given(
				tutilaction.SetBlockTime(startTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pair),
				perpaction.InsertPosition(perpaction.WithSize(sdk.ZeroDec()), perpaction.WithTrader(alice)),
				perpaction.InsertReserveSnapshot(pair, startTime, perpaction.WithPriceMultiplier(sdk.NewDec(9))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(10*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("8.5"))),
				perpaction.InsertReserveSnapshot(pair, startTime.Add(20*time.Second), perpaction.WithPriceMultiplier(sdk.MustNewDecFromStr("9.5"))),
			).
			When(
				tutilaction.MoveToNextBlockWithDuration(30 * time.Second),
			).
			Then(
				perpassert.PositionNotionalTWAPShouldBeEqualTo(pair, alice, 30*time.Second, sdk.ZeroDec()),
			),

		tutilaction.TC("single snapshot").
			Given(
				tutilaction.SetBlockTime(startTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pair),
			).
			When(
				perpaction.InsertPosition(perpaction.WithSize(sdk.NewDec(100)), perpaction.WithTrader(alice)),
				perpaction.InsertReserveSnapshot(pair, startTime, perpaction.WithPriceMultiplier(sdk.NewDec(9))),
			).
			Then(
				perpassert.PositionNotionalTWAPShouldBeEqualTo(pair, alice, 30*time.Second, sdk.MustNewDecFromStr("899.999999910000000009")),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestUnrealizedPnl(t *testing.T) {
	tests := []struct {
		name                  string
		position              types.Position
		positionNotional      sdk.Dec
		expectedUnrealizedPnl sdk.Dec
	}{
		{
			name: "long position positive pnl",
			position: types.Position{
				Size_:        sdk.NewDec(10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(15),
			expectedUnrealizedPnl: sdk.NewDec(5),
		},
		{
			name: "long position negative pnl",
			position: types.Position{
				Size_:        sdk.NewDec(10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(5),
			expectedUnrealizedPnl: sdk.NewDec(-5),
		},
		{
			name: "short position positive pnl",
			position: types.Position{
				Size_:        sdk.NewDec(-10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(5),
			expectedUnrealizedPnl: sdk.NewDec(5),
		},
		{
			name: "short position negative pnl",
			position: types.Position{
				Size_:        sdk.NewDec(-10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(15),
			expectedUnrealizedPnl: sdk.NewDec(-5),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualValues(t, tc.expectedUnrealizedPnl, keeper.UnrealizedPnl(tc.position, tc.positionNotional))
		})
	}
}

func TestMarginRatio(t *testing.T) {
	tests := []struct {
		name                string
		position            types.Position
		positionNotional    sdk.Dec
		latestCPF           sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			name: "long position, no change",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(100),
			latestCPF:           sdk.ZeroDec(),
			expectedMarginRatio: sdk.OneDec(),
		},
		{
			name: "long position, positive PnL, positive cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.981818181818181818"), // 108 / 110
		},
		{
			name: "long position, positive PnL, negative cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.018181818181818182"), // 112 / 110
		},
		{
			name: "long position, negative PnL, positive cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.977777777777777778"), // 88 / 90
		},
		{
			name: "long position, negative PnL, negative cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.OneDec(),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.022222222222222222"), // 92 / 90
		},
		{
			name: "short position, no change",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(100),
			latestCPF:           sdk.ZeroDec(),
			expectedMarginRatio: sdk.OneDec(),
		},
		{
			name: "short position, positive PnL, positive cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.244444444444444444"), // 112 / 90
		},
		{
			name: "short position, positive PnL, negative cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(90),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("1.2"), // 108 / 90
		},
		{
			name: "short position, negative PnL, positive cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.836363636363636364"), // 92 / 110
		},
		{
			name: "short position, negative PnL, negative cumulative premium fraction",
			position: types.Position{
				Margin:                          sdk.NewDec(100),
				Size_:                           sdk.NewDec(-1),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				OpenNotional:                    sdk.NewDec(100),
			},
			positionNotional:    sdk.NewDec(110),
			latestCPF:           sdk.NewDec(-2),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.8"), // 88 / 110
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualValues(t, tc.expectedMarginRatio, keeper.MarginRatio(tc.position, tc.positionNotional, tc.latestCPF))
		})
	}
}

func TestFundingPayment(t *testing.T) {
	tests := []struct {
		name                   string
		position               types.Position
		marketLatestCPF        sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name: "long position, positive cumulative premium fraction",
			position: types.Position{
				Size_:                           sdk.NewDec(10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.002"),
			expectedFundingPayment: sdk.MustNewDecFromStr("0.010"),
		},
		{
			name: "long position, negative cumulative premium fraction",
			position: types.Position{
				Size_:                           sdk.NewDec(10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.002"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.001"),
			expectedFundingPayment: sdk.MustNewDecFromStr("-0.010"),
		},
		{
			name: "short position, positive cumulative premium fraction",
			position: types.Position{
				Size_:                           sdk.NewDec(-10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.002"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.001"),
			expectedFundingPayment: sdk.MustNewDecFromStr("0.010"),
		},
		{
			name: "short position, negative cumulative premium fraction",
			position: types.Position{
				Size_:                           sdk.NewDec(-10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.002"),
			expectedFundingPayment: sdk.MustNewDecFromStr("-0.010"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fundingPayment := keeper.FundingPayment(tc.position, tc.marketLatestCPF)
			assert.EqualValues(t, tc.expectedFundingPayment, fundingPayment)
		})
	}
}

func TestMarginRatioFail(t *testing.T) {
	val := keeper.MarginRatio(types.Position{Size_: sdk.ZeroDec()}, sdk.ZeroDec(), sdk.ZeroDec())
	require.Equal(t, sdk.ZeroDec(), val)

	_, err := keeper.PositionNotionalSpot(types.AMM{PriceMultiplier: sdk.OneDec()}, types.Position{Size_: sdk.ZeroDec()})
	require.NoError(t, err)

	_, err = keeper.PositionNotionalSpot(types.AMM{}, types.Position{})
	require.ErrorContains(t, err, "input base amt is nil")
}
