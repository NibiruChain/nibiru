package keeper

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/coll"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGetMarkPrice(t *testing.T) {
	tests := []struct {
		name              string
		pair              common.AssetPair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "correctly fetch underlying price",
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(1),
			expectedPrice:     sdk.NewDec(40000),
		},
		{
			name:              "complex price",
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(2_489_723_947),
			baseAssetReserve:  sdk.NewDec(34_597_234),
			expectedPrice:     sdk.MustNewDecFromStr("71.963092396345904415"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPricefeedKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitratio=*/ sdk.OneDec(),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)

			price, err := vpoolKeeper.GetMarkPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedPrice, price)
		})
	}
}

func TestGetBaseAssetPrice(t *testing.T) {
	tests := []struct {
		name                string
		pair                common.AssetPair
		quoteAssetReserve   sdk.Dec
		baseAssetReserve    sdk.Dec
		baseAmount          sdk.Dec
		direction           types.Direction
		expectedQuoteAmount sdk.Dec
		expectedErr         error
	}{
		{
			name:                "zero base asset means zero price",
			pair:                common.Pair_BTC_NUSD,
			quoteAssetReserve:   sdk.NewDec(40_000),
			baseAssetReserve:    sdk.NewDec(10_000),
			baseAmount:          sdk.ZeroDec(),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			pair:                common.Pair_BTC_NUSD,
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:                "simple remove base from pool",
			pair:                common.Pair_BTC_NUSD,
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("1000"),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.Pair_BTC_NUSD,
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
				mock.NewMockPricefeedKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.OneDec(),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)

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
		pair               common.AssetPair
		quoteAssetReserve  sdk.Dec
		baseAssetReserve   sdk.Dec
		quoteAmount        sdk.Dec
		direction          types.Direction
		expectedBaseAmount sdk.Dec
		expectedErr        error
	}{
		{
			name:               "zero base asset means zero price",
			pair:               common.Pair_BTC_NUSD,
			quoteAssetReserve:  sdk.NewDec(40_000),
			baseAssetReserve:   sdk.NewDec(10_000),
			quoteAmount:        sdk.ZeroDec(),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.ZeroDec(),
		},
		{
			name:               "simple add base to pool",
			pair:               common.Pair_BTC_NUSD,
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:               "simple remove base from pool",
			pair:               common.Pair_BTC_NUSD,
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.Pair_BTC_NUSD,
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
				mock.NewMockPricefeedKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.OneDec(),
				sdk.OneDec(),
				sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)

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
		pair               common.AssetPair
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
		{
			name: "spot price twap calc, t=[10,30]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(90),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(85),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
		{
			name: "spot price twap calc, t=[11,35]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(90),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(85),
					time.UnixMilli(20),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
		{
			name:               "spot price twap calc, t=[0,0]",
			pair:               common.Pair_BTC_NUSD,
			reserveSnapshots:   []types.ReserveSnapshot{},
			currentBlockTime:   time.UnixMilli(0),
			currentBlockHeight: 1,
			lookbackInterval:   0 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.NewDec(100),
		},
		{
			name: "quote asset swap twap calc, add to pool, t=[10,30]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(30),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
		{
			name: "quote asset swap twap calc, remove from pool, t=[10,30]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
			name: "base asset swap twap calc, add to pool, t=[10,30]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
		{
			name: "base asset swap twap calc, remove from pool, t=[10,30]",
			pair: common.Pair_BTC_NUSD,
			reserveSnapshots: []types.ReserveSnapshot{
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					sdk.NewDec(10),
					sdk.NewDec(60),
					time.UnixMilli(10),
				),
				types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
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
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPricefeedKeeper(gomock.NewController(t)))
			ctx = ctx.WithBlockTime(time.UnixMilli(0))

			t.Log("Create an empty pool for the first block")
			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.OneDec(),
				sdk.OneDec(),
				/* maxLeverage */ sdk.NewDec(15),
			)

			t.Log("throw in another market pair to ensure key iteration doesn't overlap")
			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				sdk.ZeroDec(),
				sdk.NewDec(100),
				sdk.OneDec(),
				sdk.ZeroDec(),
				sdk.OneDec(),
				sdk.OneDec(),
				/* maxLeverage */ sdk.NewDec(15),
			)

			for _, snapshot := range tc.reserveSnapshots {
				ctx = ctx.WithBlockTime(time.UnixMilli(snapshot.TimestampMs))
				snapshot := types.NewReserveSnapshot(
					common.Pair_BTC_NUSD,
					snapshot.BaseAssetReserve,
					snapshot.QuoteAssetReserve,
					ctx.BlockTime(),
				)
				vpoolKeeper.ReserveSnapshots.Insert(ctx, coll.Join(snapshot.Pair, time.UnixMilli(snapshot.TimestampMs)), snapshot)
			}
			ctx = ctx.WithBlockTime(tc.currentBlockTime).WithBlockHeight(tc.currentBlockHeight)
			price, err := vpoolKeeper.calcTwap(ctx,
				tc.pair,
				tc.twapCalcOption,
				tc.direction,
				tc.assetAmount,
				tc.lookbackInterval,
			)
			require.NoError(t, err)
			require.EqualValuesf(t, tc.expectedPrice, price,
				"expected %s, got %s", tc.expectedPrice.String(), price.String())
		})
	}
}
