package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	keeper "github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPositionNotionalSpot(t *testing.T) {
	tests := []struct {
		name             string
		amm              *v2types.AMM
		position         v2types.Position
		expectedNotional sdk.Dec
	}{
		{
			name: "long position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: v2types.Position{
				Size_: sdk.NewDec(10),
			},
			expectedNotional: sdk.MustNewDecFromStr("19.801980198019801980"),
		},
		{
			name: "short position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: v2types.Position{
				Size_: sdk.NewDec(-10),
			},
			expectedNotional: sdk.MustNewDecFromStr("20.202020202020202020"),
		},
		{
			name: "zero position",
			amm:  mock.TestAMM(sdk.NewDec(1_000), sdk.NewDec(2)),
			position: v2types.Position{
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
	tests := []struct {
		name               string
		position           v2types.Position
		currentTimestamp   int64
		twapLookbackWindow time.Duration
		snapshots          []v2types.ReserveSnapshot
		expectedNotional   sdk.Dec
	}{
		{
			name: "long position",
			position: v2types.Position{
				Size_: sdk.NewDec(10),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			currentTimestamp:   40,
			twapLookbackWindow: 30 * time.Millisecond,
			snapshots: []v2types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.NewDec(9)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.MustNewDecFromStr("8.5")),
					TimestampMs: 20,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.MustNewDecFromStr("9.5")),
					TimestampMs: 30,
				},
			},
			expectedNotional: sdk.MustNewDecFromStr("89.999999999100000000"),
		},
		{
			name: "short position",
			position: v2types.Position{
				Size_: sdk.NewDec(-10),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			currentTimestamp:   40,
			twapLookbackWindow: 30 * time.Millisecond,
			snapshots: []v2types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.NewDec(9)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.MustNewDecFromStr("8.5")),
					TimestampMs: 20,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.MustNewDecFromStr("9.5")),
					TimestampMs: 30,
				},
			},
			expectedNotional: sdk.MustNewDecFromStr("90.000000000900000000"),
		},
		{
			name: "zero position",
			position: v2types.Position{
				Size_: sdk.ZeroDec(),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			currentTimestamp:   40,
			twapLookbackWindow: 30 * time.Millisecond,
			snapshots: []v2types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1e12), sdk.NewDec(9)),
					TimestampMs: 10,
				},
			},
			expectedNotional: sdk.ZeroDec(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			for _, s := range tc.snapshots {
				app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(s.Amm.Pair, time.UnixMilli(s.TimestampMs)), s)
			}
			ctx = ctx.WithBlockTime(time.UnixMilli(tc.currentTimestamp))

			notional, err := app.PerpKeeperV2.PositionNotionalTWAP(ctx, tc.position, tc.twapLookbackWindow)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedNotional, notional)
		})
	}
}

func TestUnrealizedPnl(t *testing.T) {
	tests := []struct {
		name                  string
		position              v2types.Position
		positionNotional      sdk.Dec
		expectedUnrealizedPnl sdk.Dec
	}{
		{
			name: "long position positive pnl",
			position: v2types.Position{
				Size_:        sdk.NewDec(10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(15),
			expectedUnrealizedPnl: sdk.NewDec(5),
		},
		{
			name: "long position negative pnl",
			position: v2types.Position{
				Size_:        sdk.NewDec(10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(5),
			expectedUnrealizedPnl: sdk.NewDec(-5),
		},
		{
			name: "short position positive pnl",
			position: v2types.Position{
				Size_:        sdk.NewDec(-10),
				OpenNotional: sdk.NewDec(10),
			},
			positionNotional:      sdk.NewDec(5),
			expectedUnrealizedPnl: sdk.NewDec(5),
		},
		{
			name: "short position negative pnl",
			position: v2types.Position{
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
		position            v2types.Position
		positionNotional    sdk.Dec
		latestCPF           sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			name: "long position, no change",
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			position: v2types.Position{
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
			marginRatio, err := keeper.MarginRatio(tc.position, tc.positionNotional, tc.latestCPF)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedMarginRatio, marginRatio)
		})
	}
}

func TestFundingPayment(t *testing.T) {
	tests := []struct {
		name                   string
		position               v2types.Position
		marketLatestCPF        sdk.Dec
		expectedFundingPayment sdk.Dec
	}{
		{
			name: "long position, positive cumulative premium fraction",
			position: v2types.Position{
				Size_:                           sdk.NewDec(10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.002"),
			expectedFundingPayment: sdk.MustNewDecFromStr("0.010"),
		},
		{
			name: "long position, negative cumulative premium fraction",
			position: v2types.Position{
				Size_:                           sdk.NewDec(10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.002"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.001"),
			expectedFundingPayment: sdk.MustNewDecFromStr("-0.010"),
		},
		{
			name: "short position, positive cumulative premium fraction",
			position: v2types.Position{
				Size_:                           sdk.NewDec(-10),
				LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.002"),
			},
			marketLatestCPF:        sdk.MustNewDecFromStr("0.001"),
			expectedFundingPayment: sdk.MustNewDecFromStr("0.010"),
		},
		{
			name: "short position, negative cumulative premium fraction",
			position: v2types.Position{
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
