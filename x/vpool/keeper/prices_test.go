package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGetMarkPrice(t *testing.T) {
	tests := []struct {
		name              string
		pair              asset.Pair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "correctly fetch underlying price",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(1),
			expectedPrice:     sdk.NewDec(40000),
		},
		{
			name:              "complex price",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve: sdk.NewDec(2_489_723_947),
			baseAssetReserve:  sdk.NewDec(34_597_234),
			expectedPrice:     sdk.MustNewDecFromStr("71.963092396345904415"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))

			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				types.VpoolConfig{
					FluctuationLimitRatio:  sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
			))

			price, err := vpoolKeeper.GetMarkPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedPrice, price)
		})
	}
}

func TestGetBaseAssetPrice(t *testing.T) {
	tests := []struct {
		name                string
		pair                asset.Pair
		quoteAssetReserve   sdk.Dec
		baseAssetReserve    sdk.Dec
		baseAmount          sdk.Dec
		direction           types.Direction
		expectedQuoteAmount sdk.Dec
		expectedErr         error
	}{
		{
			name:                "zero base asset means zero price",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve:   sdk.NewDec(40_000),
			baseAssetReserve:    sdk.NewDec(10_000),
			baseAmount:          sdk.ZeroDec(),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:                "simple remove base from pool",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("1000"),
		},
		{
			name:              "too much base removed results in error",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			baseAmount:        sdk.MustNewDecFromStr("1000"),
			direction:         types.Direction_REMOVE_FROM_POOL,
			expectedErr:       types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))

			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				types.VpoolConfig{
					FluctuationLimitRatio:  sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
			))

			quoteAmount, err := vpoolKeeper.GetBaseAssetPrice(ctx, tc.pair, tc.direction, tc.baseAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedQuoteAmount, quoteAmount,
					"expected quote: %s, got: %s", tc.expectedQuoteAmount.String(), quoteAmount.String(),
				)
			}
		})
	}
}

func TestGetQuoteAssetPrice(t *testing.T) {
	tests := []struct {
		name               string
		pair               asset.Pair
		quoteAssetReserve  sdk.Dec
		baseAssetReserve   sdk.Dec
		quoteAmount        sdk.Dec
		direction          types.Direction
		expectedBaseAmount sdk.Dec
		expectedErr        error
	}{
		{
			name:               "zero base asset means zero price",
			pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteAssetReserve:  sdk.NewDec(40_000),
			baseAssetReserve:   sdk.NewDec(10_000),
			quoteAmount:        sdk.ZeroDec(),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.ZeroDec(),
		},
		{
			name:               "simple add base to pool",
			pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:               "simple remove base from pool",
			pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much base removed results in error",
			pair:              asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			quoteAmount:       sdk.NewDec(1000),
			direction:         types.Direction_REMOVE_FROM_POOL,
			expectedErr:       types.ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))

			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				types.VpoolConfig{
					FluctuationLimitRatio:  sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
			))

			baseAmount, err := vpoolKeeper.GetQuoteAssetPrice(ctx, tc.pair, tc.direction, tc.quoteAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedBaseAmount, baseAmount,
					"expected quote: %s, got: %s", tc.expectedBaseAmount.String(), baseAmount.String(),
				)
			}
		})
	}
}

func TestCalcTwap(t *testing.T) {
	tests := []struct {
		name               string
		pair               asset.Pair
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
		// snapshot quote asset reserve at t = 0: 100
		// snapshot base asset reserve at t = 0: 1
		// expected price: ((95/10 * (35 - 30) + 85/10 * (30 - 20) + 90/10 * (20 - 10) + 100/1 * (10 - 5)) / (5 + 10 + 10 + 5)
		{
			name: "spot price twap calc, t=[5,35]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(90),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(85),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(95),
					time.UnixMilli(30),
				),
			},
			currentBlockTime:   time.UnixMilli(35),
			currentBlockHeight: 3,
			lookbackInterval:   30 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("24.083333333333333333"),
		},

		// expected price: (95/10 * (30 - 30) + 85/10 * (30 - 20) + 90/10 * (20 - 10)) / (10 + 10)
		{
			name: "spot price twap calc, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(90),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(85),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(95),
					time.UnixMilli(30),
				),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(90),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(85),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(95),
					time.UnixMilli(30),
				),
			},
			currentBlockTime:   time.UnixMilli(35),
			currentBlockHeight: 4,
			lookbackInterval:   24 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("8.895833333333333333"),
		},

		// base asset reserve at t = 0: 1
		// quote asset reserve at t = 0: 100
		// expected price: 100/1
		{
			name:               "spot price twap calc, t=[0,0]",
			pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots:   []types.ReserveSnapshot{},
			currentBlockTime:   time.UnixMilli(0),
			currentBlockHeight: 1,
			lookbackInterval:   0 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.NewDec(100),
		},

		// k: 30 * 100 = 300
		// asset amount : 10
		// expected price: ((7.5 - 300/(40 + 10)) * (30 - 20) + (10 - 300/(30 + 10)) * (20 - 10)) / (10 + 10)
		{
			name: "quote asset swap twap calc, add to pool, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(30),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.MustNewDecFromStr("7.5"),
					sdk.NewDec(40),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_ADD_TO_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.NewDec(2),
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((12 - 600/(50 - 10)) * (30 - 20) + (10 - 600/(60 - 10)) * (20 - 10)) / (10 + 10)
		{
			name: "quote asset swap twap calc, remove from pool, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(12),
					sdk.NewDec(50),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_REMOVE_FROM_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.MustNewDecFromStr("2.5"),
		},
		{
			name: "Error: quote asset reserve = asset amount",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(20),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_REMOVE_FROM_POOL,
			assetAmount:        sdk.NewDec(20),
			expectedErr:        types.ErrQuoteReserveAtZero,
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((60 - 600/(10 + 10)) * (20 - 10) + (30 - 600/(20 + 10)) * (30 - 20)) / (10 + 10)
		{
			name: "base asset swap twap calc, add to pool, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(20),
					sdk.NewDec(30),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_ADD_TO_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.NewDec(20),
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((60 - 600/(10 - 2)) * (20 - 10) + (75 - 600/(8 - 2)) * (30 - 20)) / (10 + 10)
		{
			name: "base asset swap twap calc, remove from pool, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(8),
					sdk.NewDec(75),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_REMOVE_FROM_POOL,
			assetAmount:        sdk.NewDec(2),
			expectedPrice:      sdk.NewDec(20),
		},
		{
			name: "Error: base asset reserve = asset amount",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_REMOVE_FROM_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedErr:        types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))
			ctx = ctx.WithBlockTime(time.UnixMilli(0))

			t.Log("Create an empty pool for the first block")
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				/*quoteAssetReserve=*/ sdk.OneDec(),
				/*baseAssetReserve=*/ sdk.OneDec(),
				types.DefaultVpoolConfig().WithMaxLeverage(sdk.NewDec(15)),
			))

			t.Log("throw in another market pair to ensure key iteration doesn't overlap")
			assert.NoError(t, vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				/*quoteAssetReserve=*/ sdk.NewDec(100),
				/*baseAssetReserve=*/ sdk.OneDec(),
				types.DefaultVpoolConfig().WithMaxLeverage(sdk.NewDec(15)),
			))

			for _, snapshot := range tc.reserveSnapshots {
				ctx = ctx.WithBlockTime(time.UnixMilli(snapshot.TimestampMs))
				snapshot := types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					snapshot.BaseAssetReserve,
					snapshot.QuoteAssetReserve,
					ctx.BlockTime(),
				)
				vpoolKeeper.ReserveSnapshots.Insert(ctx, collections.Join(snapshot.Pair, time.UnixMilli(snapshot.TimestampMs)), snapshot)
			}
			ctx = ctx.WithBlockTime(tc.currentBlockTime).WithBlockHeight(tc.currentBlockHeight)
			price, err := vpoolKeeper.calcTwap(ctx,
				tc.pair,
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
