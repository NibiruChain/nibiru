package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func TestCalcTwap(t *testing.T) {
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
			Then(
				TwapShouldBe(pairBtcUsdc, types.TwapCalcOption_SPOT, types.Direction_DIRECTION_UNSPECIFIED, sdk.ZeroDec(), 30*time.Second, sdk.NewDec(10)),
			),

		TC("base asset twap, long").
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
			Then(
				TwapShouldBe(pairBtcUsdc, types.TwapCalcOption_BASE_ASSET_SWAP, types.Direction_LONG, sdk.NewDec(5), 30*time.Second, sdk.MustNewDecFromStr("50.000000000250000000")),
			),

		TC("base asset twap, short").
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
			Then(
				TwapShouldBe(pairBtcUsdc, types.TwapCalcOption_BASE_ASSET_SWAP, types.Direction_SHORT, sdk.NewDec(5), 30*time.Second, sdk.MustNewDecFromStr("49.999999999750000000")),
			),

		TC("quote asset twap, long").
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
			Then(
				TwapShouldBe(pairBtcUsdc, types.TwapCalcOption_QUOTE_ASSET_SWAP, types.Direction_LONG, sdk.NewDec(5), 30*time.Second, sdk.MustNewDecFromStr("0.503367003366748282")),
			),

		TC("quote asset twap, short").
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
			Then(
				TwapShouldBe(pairBtcUsdc, types.TwapCalcOption_QUOTE_ASSET_SWAP, types.Direction_SHORT, sdk.NewDec(5), 30*time.Second, sdk.MustNewDecFromStr("0.503367003367258451")),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestInvalidTwap(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, _ := testapp.NewNibiruTestAppAndContext()
	ctx := app.NewContext(false, tmproto.Header{
		Height: 1,
	})
	startTime := time.UnixMilli(0)

	app.PerpKeeperV2.SaveMarket(ctx, *mock.TestMarket())
	app.PerpKeeperV2.SaveAMM(ctx, *mock.TestAMMDefault())
	app.PerpKeeperV2.ReserveSnapshots.Insert(
		ctx, collections.Join(pair, startTime), types.ReserveSnapshot{
			Amm:         *mock.TestAMMDefault(),
			TimestampMs: 0,
		})

	tc := struct {
		twapCalcOption   types.TwapCalcOption
		direction        types.Direction
		assetAmount      sdk.Dec
		lookbackInterval time.Duration
	}{
		twapCalcOption:   types.TwapCalcOption_SPOT,
		direction:        types.Direction_DIRECTION_UNSPECIFIED,
		assetAmount:      sdk.ZeroDec(),
		lookbackInterval: 30 * time.Second,
	}
	_, err := app.PerpKeeperV2.CalcTwap(ctx,
		pair,
		tc.twapCalcOption,
		tc.direction,
		tc.assetAmount,
		tc.lookbackInterval,
	)
	require.ErrorIs(t, err, types.ErrNoValidTWAP)
}

func TestCalcTwapExtended(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := []struct {
		name               string
		reserveSnapshots   []types.ReserveSnapshot
		currentBlockTime   time.Time
		currentBlockHeight int64
		lookbackInterval   time.Duration
		twapCalcOption     types.TwapCalcOption
		direction          types.Direction
		assetAmount        sdk.Dec
		expectedPrice      sdk.Dec
		expectedErr        error
	}{
		// Same price at 9 for 20 milliseconds
		{
			name: "spot price twap calc, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.NewDec(9)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.NewDec(9)),
					TimestampMs: 20,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.NewDec(9)),
					TimestampMs: 30,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("9"),
		},
		// expected price: (9.5 * (30 - 30) + 8.5 * (30 - 20) + 9 * (20 - 10)) / (10 + 10)
		{
			name: "spot price twap calc, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.NewDec(9)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.MustNewDecFromStr("8.5")),
					TimestampMs: 20,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.MustNewDecFromStr("9.5")),
					TimestampMs: 30,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("8.75"),
		},
		// expected price: (95/10 * (35 - 30) + 85/10 * (30 - 20) + 90/10 * (20 - 11)) / (5 + 10 + 9)
		{
			name: "spot price twap calc, t=[11,35]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.NewDec(9)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.MustNewDecFromStr("8.5")),
					TimestampMs: 20,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(100), sdk.MustNewDecFromStr("9.5")),
					TimestampMs: 30,
				},
			},
			currentBlockTime:   time.UnixMilli(35),
			currentBlockHeight: 4,
			lookbackInterval:   24 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("8.895833333333333333"),
		},

		// base asset reserve at t = 0: 100
		// quote asset reserve at t = 0: 100
		// expected price: 1
		{
			name:               "spot price twap calc, t=[0,0]",
			reserveSnapshots:   []types.ReserveSnapshot{},
			currentBlockTime:   time.UnixMilli(0),
			currentBlockHeight: 1,
			lookbackInterval:   0 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.OneDec(),
		},

		// expected price: (95/10 * (35 - 30) + 85/10 * (30 - 20) + 90/10 * (20 - 11)) / (5 + 10 + 9)
		{
			name: "quote asset swap twap calc, add to pool, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(3)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(5)),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_LONG,
			assetAmount:        sdk.NewDec(5),
			expectedPrice:      sdk.MustNewDecFromStr("1.331447254908153411"), // ~ 5 base at a twap price of 4
		},

		// expected price: (95/10 * (35 - 30) + 85/10 * (30 - 20) + 90/10 * (20 - 11)) / (5 + 10 + 9)
		{
			name: "quote asset swap twap calc, remove from pool, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(3)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(5)),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_SHORT,
			assetAmount:        sdk.NewDec(5),
			expectedPrice:      sdk.MustNewDecFromStr("1.335225041402003005"), // ~ 5 base at a twap price of 4
		},

		{
			name: "Error: quote asset reserve = asset amount",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(10), sdk.NewDec(2)),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_SHORT,
			assetAmount:        sdk.NewDec(20),
			expectedErr:        types.ErrQuoteReserveAtZero,
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((60 - 600/(10 + 10)) * (20 - 10) + (30 - 600/(20 + 10)) * (30 - 20)) / (10 + 10)
		{
			name: "base asset swap twap calc, add to pool, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(6)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.MustNewDecFromStr("1.5")),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_SHORT,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.MustNewDecFromStr("37.128712871287128712"),
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((60 - 600/(10 - 2)) * (20 - 10) + (75 - 600/(8 - 2)) * (30 - 20)) / (10 + 10)
		{
			name: "base asset swap twap calc, remove from pool, t=[10,30]",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.NewDec(6)),
					TimestampMs: 10,
				},
				{
					Amm:         *mock.TestAMM(sdk.NewDec(1000), sdk.MustNewDecFromStr("9.375")),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_LONG,
			assetAmount:        sdk.NewDec(2),
			expectedPrice:      sdk.MustNewDecFromStr("15.405811623246492984"),
		},
		{
			name: "Error: base asset reserve = asset amount",
			reserveSnapshots: []types.ReserveSnapshot{
				{
					Amm:         *mock.TestAMM(sdk.NewDec(10), sdk.NewDec(9)),
					TimestampMs: 20,
				},
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_LONG,
			assetAmount:        sdk.NewDec(10),
			expectedErr:        types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, _ := testapp.NewNibiruTestAppAndContext()
			ctx := app.NewContext(false, tmproto.Header{
				Height: 1,
			})
			startTime := time.UnixMilli(0)

			app.PerpKeeperV2.SaveMarket(ctx, *mock.TestMarket())
			app.PerpKeeperV2.SaveAMM(ctx, *mock.TestAMMDefault())
			app.PerpKeeperV2.ReserveSnapshots.Insert(
				ctx, collections.Join(pair, startTime), types.ReserveSnapshot{
					Amm:         *mock.TestAMMDefault(),
					TimestampMs: 0,
				})

			for _, snapshot := range tc.reserveSnapshots {
				ctx = ctx.WithBlockTime(time.UnixMilli(snapshot.TimestampMs))
				app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(snapshot.Amm.Pair, time.UnixMilli(snapshot.TimestampMs)), snapshot)
			}
			ctx = ctx.WithBlockTime(tc.currentBlockTime).WithBlockHeight(tc.currentBlockHeight)

			price, err := app.PerpKeeperV2.CalcTwap(ctx,
				pair,
				tc.twapCalcOption,
				tc.direction,
				tc.assetAmount,
				tc.lookbackInterval,
			)
			require.ErrorIs(t, err, tc.expectedErr)
			require.EqualValuesf(t, tc.expectedPrice, price,
				"expected %s, got %s", tc.expectedPrice.String(), price.String())
		})
	}
}
