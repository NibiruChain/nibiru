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
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func TestGetMarkPrice(t *testing.T) {
	tests := []struct {
		name          string
		pair          asset.Pair
		quoteReserve  sdk.Dec
		baseReserve   sdk.Dec
		pegMultiplier sdk.Dec
		expectedPrice sdk.Dec
	}{
		{
			name:          "correctly fetch underlying price",
			pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:  sdk.NewDec(1),
			baseReserve:   sdk.NewDec(1),
			pegMultiplier: sdk.NewDec(40_000),
			expectedPrice: sdk.NewDec(40000),
		},
		{
			name:          "complex price",
			pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:  sdk.NewDec(34_597_234),
			baseReserve:   sdk.NewDec(34_597_234),
			pegMultiplier: sdk.MustNewDecFromStr("71.963092396345904415"),
			expectedPrice: sdk.MustNewDecFromStr("71.963092396345904415"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpammKeeper, ctx := PerpAmmKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))

			require.NoError(t, perpammKeeper.CreatePool(
				ctx,
				tc.pair,
				tc.quoteReserve,
				tc.baseReserve,
				types.MarketConfig{
					FluctuationLimitRatio:  sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
				tc.pegMultiplier,
			))

			price, err := perpammKeeper.GetMarkPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedPrice, price)
		})
	}
}

func TestGetBaseAssetPrice(t *testing.T) {
	tests := []struct {
		name                string
		pair                asset.Pair
		quoteReserve        sdk.Dec
		baseReserve         sdk.Dec
		baseAmount          sdk.Dec
		pegMultiplier       sdk.Dec
		direction           types.Direction
		expectedQuoteAmount sdk.Dec
		expectedErr         error
	}{
		{
			name:                "zero base asset means zero price",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			quoteReserve:        sdk.NewDec(10_000),
			baseReserve:         sdk.NewDec(10_000),
			baseAmount:          sdk.ZeroDec(),
			pegMultiplier:       sdk.NewDec(4),
			direction:           types.Direction_LONG,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseReserve:         sdk.NewDec(1000),
			quoteReserve:        sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			pegMultiplier:       sdk.OneDec(),
			direction:           types.Direction_LONG,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:                "simple remove base from pool",
			pair:                asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseReserve:         sdk.NewDec(1000),
			quoteReserve:        sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			pegMultiplier:       sdk.OneDec(),
			direction:           types.Direction_SHORT,
			expectedQuoteAmount: sdk.MustNewDecFromStr("1000"),
		},
		{
			name:          "too much base removed results in error",
			pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			baseReserve:   sdk.NewDec(1000),
			quoteReserve:  sdk.NewDec(1000),
			pegMultiplier: sdk.OneDec(),
			baseAmount:    sdk.MustNewDecFromStr("1000"),
			direction:     types.Direction_SHORT,
			expectedErr:   types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpammKeeper, ctx := PerpAmmKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))

			assert.NoError(t, perpammKeeper.CreatePool(
				ctx,
				tc.pair,
				tc.quoteReserve,
				tc.baseReserve,
				types.MarketConfig{
					FluctuationLimitRatio:  sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					TradeLimitRatio:        sdk.OneDec(),
				},
				tc.pegMultiplier,
			))

			market, err := perpammKeeper.GetPool(ctx, tc.pair)
			require.NoError(t, err)

			quoteAmount, err := perpammKeeper.GetBaseAssetPrice(market, tc.direction, tc.baseAmount)
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
		// Same price at 9 for 20 blocks
		{
			name: "spot price twap calc, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(30),
				),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("8.5"),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("9.5"),
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
			name: "spot price twap calc, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("8.5"),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(100),
					/* Quote asset reserve */ sdk.NewDec(100),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("9.5"),
					time.UnixMilli(30),
				),
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
			pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(3),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(5),
					time.UnixMilli(20),
				),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(3),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(5),
					time.UnixMilli(20),
				),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(10),
					/* Quote asset reserve */ sdk.NewDec(10),
					/* Peg multiplier*/ sdk.NewDec(2),
					time.UnixMilli(20),
				),
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
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(6),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("1.5"),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_LONG,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.MustNewDecFromStr("37.128712871287128712"),
		},

		// k: 60 * 100 = 600
		// asset amount: 10
		// expected price: ((60 - 600/(10 - 2)) * (20 - 10) + (75 - 600/(8 - 2)) * (30 - 20)) / (10 + 10)
		{
			name: "base asset swap twap calc, remove from pool, t=[10,30]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.NewDec(6),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(1_000),
					/* Quote asset reserve */ sdk.NewDec(1_000),
					/* Peg multiplier*/ sdk.MustNewDecFromStr("9.375"),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_SHORT,
			assetAmount:        sdk.NewDec(2),
			expectedPrice:      sdk.MustNewDecFromStr("15.405811623246492984"),
		},
		{
			name: "Error: base asset reserve = asset amount",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					/* Pair */ asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					/* Base asset reserve */ sdk.NewDec(10),
					/* Quote asset reserve */ sdk.NewDec(10),
					/* Peg multiplier*/ sdk.NewDec(9),
					time.UnixMilli(20),
				),
			},
			currentBlockTime:   time.UnixMilli(30),
			currentBlockHeight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_SHORT,
			assetAmount:        sdk.NewDec(10),
			expectedErr:        types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpammKeeper, ctx := PerpAmmKeeper(t,
				mock.NewMockOracleKeeper(gomock.NewController(t)))
			ctx = ctx.WithBlockTime(time.UnixMilli(0))

			t.Log("Create an empty pool for the first block")
			assert.NoError(t, perpammKeeper.CreatePool(
				ctx,
				tc.pair,
				/* Base asset reserve */ sdk.NewDec(100),
				/* Quote asset reserve */ sdk.NewDec(100),
				*types.DefaultMarketConfig().WithMaxLeverage(sdk.NewDec(15)),
				sdk.OneDec(),
			))

			t.Log("throw in another market pair to ensure key iteration doesn't overlap")
			assert.NoError(t, perpammKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				/*quoteReserve=*/ sdk.NewDec(100),
				/*baseReserve=*/ sdk.NewDec(100),
				*types.DefaultMarketConfig().WithMaxLeverage(sdk.NewDec(15)),
				sdk.OneDec(),
			))

			for _, snapshot := range tc.reserveSnapshots {
				ctx = ctx.WithBlockTime(time.UnixMilli(snapshot.TimestampMs))
				snapshot := types.NewReserveSnapshot(
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					snapshot.BaseReserve,
					snapshot.QuoteReserve,
					snapshot.PegMultiplier,
					ctx.BlockTime(),
				)
				perpammKeeper.ReserveSnapshots.Insert(ctx, collections.Join(snapshot.Pair, time.UnixMilli(snapshot.TimestampMs)), snapshot)
			}
			ctx = ctx.WithBlockTime(tc.currentBlockTime).WithBlockHeight(tc.currentBlockHeight)
			price, err := perpammKeeper.calcTwap(ctx,
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
