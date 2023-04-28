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

// func TestGetPositionNotionalAndUnrealizedPnl(t *testing.T) {
// 	tests := []struct {
// 		name                       string
// 		initialPosition            v2types.Position
// 		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
// 		pnlCalcOption              types.PnLCalcOption
// 		expectedPositionalNotional sdk.Dec
// 		expectedUnrealizedPnL      sdk.Dec
// 	}{
// 		{
// 			name: "long position; positive pnl; spot price calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(10),
// 		},
// 		{
// 			name: "long position; negative pnl; spot price calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(5), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(-5),
// 		},
// 		{
// 			name: "long position; positive pnl; twap calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(20), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_TWAP,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(10),
// 		},
// 		{
// 			name: "long position; negative pnl; twap calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(5), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_TWAP,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(-5),
// 		},
// 		{
// 			name: "long position; positive pnl; oracle calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockOracleKeeper.EXPECT().
// 					GetExchangeRate(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 					).
// 					Return(sdk.NewDec(2), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_ORACLE,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(10),
// 		},
// 		{
// 			name: "long position; negative pnl; oracle calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockOracleKeeper.EXPECT().
// 					GetExchangeRate(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 					).
// 					Return(sdk.MustNewDecFromStr("0.5"), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_ORACLE,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(-5),
// 		},
// 		{
// 			name: "short position; positive pnl; spot price calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_SHORT,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(5), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(5),
// 		},
// 		{
// 			name: "short position; negative pnl; spot price calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_SHORT,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(-10),
// 		},
// 		{
// 			name: "short position; positive pnl; twap calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_SHORT,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(5), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_TWAP,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(5),
// 		},
// 		{
// 			name: "short position; negative pnl; twap calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_SHORT,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(20), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_TWAP,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(-10),
// 		},
// 		{
// 			name: "short position; positive pnl; oracle calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockOracleKeeper.EXPECT().
// 					GetExchangeRate(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 					).
// 					Return(sdk.MustNewDecFromStr("0.5"), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_ORACLE,
// 			expectedPositionalNotional: sdk.NewDec(5),
// 			expectedUnrealizedPnL:      sdk.NewDec(5),
// 		},
// 		{
// 			name: "long position; negative pnl; oracle calc",
// 			initialPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(-10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				mocks.mockOracleKeeper.EXPECT().
// 					GetExchangeRate(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 					).
// 					Return(sdk.NewDec(2), nil)
// 			},
// 			pnlCalcOption:              types.PnLCalcOption_ORACLE,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnL:      sdk.NewDec(-10),
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			perpKeeper, mocks, ctx := getKeeper(t)

// 			tc.setMocks(ctx, mocks)

// 			market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 			positionalNotional, unrealizedPnl, err := perpKeeper.
// 				PositionNotional(
// 					ctx,
// 					market,
// 					*mock.TestAMMDefault(),
// 					tc.initialPosition,
// 					tc.pnlCalcOption,
// 				)
// 			require.NoError(t, err)

// 			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
// 			assert.EqualValues(t, tc.expectedUnrealizedPnL, unrealizedPnl)
// 		})
// 	}
// }

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
			require.Nil(t, err)
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
			require.Nil(t, err)
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

			require.Nil(t, err)
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

// func TestGetPreferencePositionNotionalAndUnrealizedPnL(t *testing.T) {
// 	// all tests are assumed long positions with positive pnl for ease of calculation
// 	// short positions and negative pnl are implicitly correct because of
// 	// TestGetPositionNotionalAndUnrealizedPnl
// 	testcases := []struct {
// 		name                       string
// 		initPosition               v2types.Position
// 		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
// 		pnlPreferenceOption        types.PnLPreferenceOption
// 		expectedPositionalNotional sdk.Dec
// 		expectedUnrealizedPnl      sdk.Dec
// 	}{
// 		{
// 			name: "max pnl, pick spot price",
// 			initPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				t.Log("Mock market spot price")
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 				t.Log("Mock market twap")
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(15), nil)
// 			},
// 			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnl:      sdk.NewDec(10),
// 		},
// 		{
// 			name: "max pnl, pick twap",
// 			initPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				t.Log("Mock market spot price")
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 				t.Log("Mock market twap")
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(30), nil)
// 			},
// 			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
// 			expectedPositionalNotional: sdk.NewDec(30),
// 			expectedUnrealizedPnl:      sdk.NewDec(20),
// 		},
// 		{
// 			name: "min pnl, pick spot price",
// 			initPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				t.Log("Mock market spot price")
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 				t.Log("Mock market twap")
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(30), nil)
// 			},
// 			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
// 			expectedPositionalNotional: sdk.NewDec(20),
// 			expectedUnrealizedPnl:      sdk.NewDec(10),
// 		},
// 		{
// 			name: "min pnl, pick twap",
// 			initPosition: v2types.Position{
// 				TraderAddress: testutilevents.AccAddress().String(),
// 				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:         sdk.NewDec(10),
// 				OpenNotional:  sdk.NewDec(10),
// 				Margin:        sdk.NewDec(1),
// 			},
// 			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
// 				t.Log("Mock market spot price")
// 				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetPrice(
// 						market,
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 					).
// 					Return(sdk.NewDec(20), nil)
// 				t.Log("Mock market twap")
// 				mocks.mockPerpAmmKeeper.EXPECT().
// 					GetBaseAssetTWAP(
// 						ctx,
// 						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 						v2types.Direction_LONG,
// 						sdk.NewDec(10),
// 						15*time.Minute,
// 					).
// 					Return(sdk.NewDec(15), nil)
// 			},
// 			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
// 			expectedPositionalNotional: sdk.NewDec(15),
// 			expectedUnrealizedPnl:      sdk.NewDec(5),
// 		},
// 	}

// 	for _, tc := range testcases {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			perpKeeper, mocks, ctx := getKeeper(t)

// 			tc.setMocks(ctx, mocks)

// 			market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
// 			positionalNotional, unrealizedPnl, err := perpKeeper.
// 				GetPreferencePositionNotionalAndUnrealizedPnL(
// 					ctx,
// 					market,
// 					*mock.TestAMMDefault(),
// 					tc.initPosition,
// 					tc.pnlPreferenceOption,
// 				)

// 			require.NoError(t, err)
// 			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
// 			assert.EqualValues(t, tc.expectedUnrealizedPnl, unrealizedPnl)
// 		})
// 	}
// }
