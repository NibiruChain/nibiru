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

func TestPositionNotionalOracle(t *testing.T) {
	tests := []struct {
		name             string
		position         v2types.Position
		oraclePrice      sdk.Dec
		expectedNotional sdk.Dec
	}{
		{
			name: "long position",
			position: v2types.Position{
				Size_: sdk.NewDec(10),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			oraclePrice:      sdk.NewDec(9),
			expectedNotional: sdk.NewDec(90),
		},
		{
			name: "short position",
			position: v2types.Position{
				Size_: sdk.NewDec(-10),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			oraclePrice:      sdk.NewDec(9),
			expectedNotional: sdk.NewDec(90),
		},
		{
			name: "zero position",
			position: v2types.Position{
				Size_: sdk.ZeroDec(),
				Pair:  asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			},
			oraclePrice:      sdk.NewDec(9),
			expectedNotional: sdk.ZeroDec(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)

			app.OracleKeeper.SetPrice(ctx, tc.position.Pair, tc.oraclePrice)

			notional, err := app.PerpKeeperV2.PositionNotionalOracle(ctx, tc.position)

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
