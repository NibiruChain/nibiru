package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGetUnderlyingPrice(t *testing.T) {
	tests := []struct {
		name           string
		pair           common.AssetPair
		pricefeedPrice sdk.Dec
	}{
		{
			name:           "correctly fetch underlying price",
			pair:           common.PairBTCStable,
			pricefeedPrice: sdk.NewDec(40000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockPricefeedKeeper := mock.NewMockPricefeedKeeper(gomock.NewController(t))
			vpoolKeeper, ctx := VpoolKeeper(t, mockPricefeedKeeper)

			mockPricefeedKeeper.
				EXPECT().
				GetCurrentPrice(
					gomock.Eq(ctx),
					gomock.Eq(tc.pair.BaseDenom()),
					gomock.Eq(tc.pair.QuoteDenom()),
				).
				Return(
					pftypes.CurrentPrice{
						PairID: tc.pair.String(),
						Price:  tc.pricefeedPrice,
					}, nil,
				)

			price, err := vpoolKeeper.GetUnderlyingPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.pricefeedPrice, price)
		})
	}
}

func TestGetSpotPrice(t *testing.T) {
	tests := []struct {
		name              string
		pair              common.AssetPair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "correctly fetch underlying price",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(1),
			expectedPrice:     sdk.NewDec(40000),
		},
		{
			name:              "complex price",
			pair:              common.PairBTCStable,
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
			)

			price, err := vpoolKeeper.GetSpotPrice(ctx, tc.pair)
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
			pair:                common.PairBTCStable,
			quoteAssetReserve:   sdk.NewDec(40_000),
			baseAssetReserve:    sdk.NewDec(10_000),
			baseAmount:          sdk.ZeroDec(),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			pair:                common.PairBTCStable,
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:                "simple remove base from pool",
			pair:                common.PairBTCStable,
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("1000"),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.PairBTCStable,
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
			pair:               common.PairBTCStable,
			quoteAssetReserve:  sdk.NewDec(40_000),
			baseAssetReserve:   sdk.NewDec(10_000),
			quoteAmount:        sdk.ZeroDec(),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.ZeroDec(),
		},
		{
			name:               "simple add base to pool",
			pair:               common.PairBTCStable,
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:               "simple remove base from pool",
			pair:               common.PairBTCStable,
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.PairBTCStable,
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
		currentBlocktime   time.Time
		currentBlockheight int64
		lookbackInterval   time.Duration
		twapCalcOption     types.TwapCalcOption
		direction          types.Direction
		assetAmount        sdk.Dec
		expectedPrice      sdk.Dec
		expectedErr        error
	}{
		{
			name: "spot price twap calc, t=[10,30]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(90),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(85),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       20,
					BlockNumber:       2,
				},
				{
					QuoteAssetReserve: sdk.NewDec(95),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       30,
					BlockNumber:       3,
				},
			},
			currentBlocktime:   time.UnixMilli(30),
			currentBlockheight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("8.75"),
		},
		{
			name: "spot price twap calc, t=[11,35]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(90),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(85),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       20,
					BlockNumber:       2,
				},
				{
					QuoteAssetReserve: sdk.NewDec(95),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       30,
					BlockNumber:       3,
				},
			},
			currentBlocktime:   time.UnixMilli(35),
			currentBlockheight: 4,
			lookbackInterval:   24 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_SPOT,
			expectedPrice:      sdk.MustNewDecFromStr("8.895833333333333333"),
		},
		{
			name: "quote asset swap twap calc, add to pool, t=[10,30]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(30),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(40),
					BaseAssetReserve:  sdk.MustNewDecFromStr("7.5"),
					TimestampMs:       20,
					BlockNumber:       2,
				},
			},
			currentBlocktime:   time.UnixMilli(30),
			currentBlockheight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_ADD_TO_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.NewDec(2),
		},
		{
			name: "quote asset swap twap calc, remove from pool, t=[10,30]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(60),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(50),
					BaseAssetReserve:  sdk.NewDec(12),
					TimestampMs:       20,
					BlockNumber:       2,
				},
			},
			currentBlocktime:   time.UnixMilli(30),
			currentBlockheight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:          types.Direction_REMOVE_FROM_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.MustNewDecFromStr("2.5"),
		},
		{
			name: "base asset swap twap calc, add to pool, t=[10,30]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(60),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(30),
					BaseAssetReserve:  sdk.NewDec(20),
					TimestampMs:       20,
					BlockNumber:       2,
				},
			},
			currentBlocktime:   time.UnixMilli(30),
			currentBlockheight: 3,
			lookbackInterval:   20 * time.Millisecond,
			twapCalcOption:     types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:          types.Direction_ADD_TO_POOL,
			assetAmount:        sdk.NewDec(10),
			expectedPrice:      sdk.NewDec(20),
		},
		{
			name: "base asset swap twap calc, remove from pool, t=[10,30]",
			pair: common.PairBTCStable,
			reserveSnapshots: []types.ReserveSnapshot{
				{
					QuoteAssetReserve: sdk.NewDec(60),
					BaseAssetReserve:  sdk.NewDec(10),
					TimestampMs:       10,
					BlockNumber:       1,
				},
				{
					QuoteAssetReserve: sdk.NewDec(75),
					BaseAssetReserve:  sdk.NewDec(8),
					TimestampMs:       20,
					BlockNumber:       2,
				},
			},
			currentBlocktime:   time.UnixMilli(30),
			currentBlockheight: 3,
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
			ctx = ctx.WithBlockTime(time.UnixMilli(0)).WithBlockHeight(0)

			t.Log("Create an empty pool for the first block, it's snapshot won't be used")
			vpoolKeeper.CreatePool(
				ctx,
				tc.pair,
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.ZeroDec(),
				sdk.OneDec(),
				sdk.OneDec(),
			)

			for i, snapshot := range tc.reserveSnapshots {
				ctx = ctx.WithBlockHeight(snapshot.BlockNumber).WithBlockTime(time.UnixMilli(snapshot.TimestampMs))
				vpoolKeeper.saveSnapshot(
					ctx,
					tc.pair,
					uint64(i+1),
					snapshot.QuoteAssetReserve,
					snapshot.BaseAssetReserve,
				)
			}
			vpoolKeeper.saveSnapshotCounter(ctx, tc.pair, uint64(len(tc.reserveSnapshots)))
			ctx = ctx.WithBlockTime(tc.currentBlocktime).WithBlockHeight(tc.currentBlockheight)

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

func TestGetTWAP(t *testing.T) {
	type positionUpdate struct {
		quoteAsset sdk.Dec
		baseAsset  sdk.Dec
		blockTs    time.Time
		direction  types.Direction
	}
	tests := []struct {
		name            string
		pair            common.AssetPair
		positionUpdates []positionUpdate

		expectedTWAPs      []sdk.Dec
		expectedMarkPrices []sdk.Dec
	}{
		{
			name:               "Add quote to position",
			pair:               common.PairBTCStable,
			positionUpdates:    []positionUpdate{{quoteAsset: sdk.NewDec(5_000), direction: types.Direction_ADD_TO_POOL, blockTs: time.Unix(2, 0)}},
			expectedTWAPs:      []sdk.Dec{sdk.MustNewDecFromStr("40006.667083333333333336")},
			expectedMarkPrices: []sdk.Dec{sdk.MustNewDecFromStr("40010.000625000000000004")},
		}, {
			name:               "Remove quote from position",
			pair:               common.PairBTCStable,
			positionUpdates:    []positionUpdate{{quoteAsset: sdk.NewDec(4_000), direction: types.Direction_REMOVE_FROM_POOL, blockTs: time.Unix(2, 0)}},
			expectedTWAPs:      []sdk.Dec{sdk.MustNewDecFromStr("39994.666933333333333333")},
			expectedMarkPrices: []sdk.Dec{sdk.MustNewDecFromStr("39992.000400000000000000")},
		}, {
			name: "Add and remove to/from quote position to return to initial TWAP",
			pair: common.PairBTCStable,
			positionUpdates: []positionUpdate{
				{quoteAsset: sdk.NewDec(700), direction: types.Direction_ADD_TO_POOL, blockTs: time.Unix(4, 0)},
				{quoteAsset: sdk.NewDec(1_234), direction: types.Direction_REMOVE_FROM_POOL, blockTs: time.Unix(7, 0)},
			},
			expectedTWAPs: []sdk.Dec{
				sdk.MustNewDecFromStr("40001.120009799999999993"),
				sdk.MustNewDecFromStr("39999.843674908525000000"),
			},
			expectedMarkPrices: []sdk.Dec{
				sdk.MustNewDecFromStr("40001.400012249999999991"),
				sdk.MustNewDecFromStr("39998.932007128900000006"),
			},
		}, {
			name:               "Add base to position",
			pair:               common.PairBTCStable,
			positionUpdates:    []positionUpdate{{baseAsset: sdk.NewDec(50), direction: types.Direction_ADD_TO_POOL, blockTs: time.Unix(2, 0)}},
			expectedTWAPs:      []sdk.Dec{sdk.MustNewDecFromStr("37520.786092214663643235")},
			expectedMarkPrices: []sdk.Dec{sdk.MustNewDecFromStr("36281.179138321995464853")}},
		{
			name:               "Remove base from position",
			pair:               common.PairBTCStable,
			positionUpdates:    []positionUpdate{{baseAsset: sdk.NewDec(40), direction: types.Direction_REMOVE_FROM_POOL, blockTs: time.Unix(2, 0)}},
			expectedTWAPs:      []sdk.Dec{sdk.MustNewDecFromStr("42268.518518518518518519")},
			expectedMarkPrices: []sdk.Dec{sdk.MustNewDecFromStr("43402.777777777777777778")},
		},
		{
			name: "Add and remove to/from base position to return to initial TWAP",
			pair: common.PairBTCStable,
			positionUpdates: []positionUpdate{
				{baseAsset: sdk.NewDec(7), direction: types.Direction_ADD_TO_POOL, blockTs: time.Unix(4, 0)},
				{baseAsset: sdk.MustNewDecFromStr("1.234"), direction: types.Direction_REMOVE_FROM_POOL, blockTs: time.Unix(7, 0)},
			},
			expectedTWAPs: []sdk.Dec{
				sdk.MustNewDecFromStr("39556.660476959200196440"),
				sdk.MustNewDecFromStr("39548.504707649598914130"),
			},
			expectedMarkPrices: []sdk.Dec{
				sdk.MustNewDecFromStr("39445.825596199000245550"),
				sdk.MustNewDecFromStr("39542.679158142740855338"),
			},
		},
	}

	initialTWAP := sdk.NewDec(40_000)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pfKeeper := mock.NewMockPricefeedKeeper(gomock.NewController(t))
			pfKeeper.EXPECT().IsActivePair(gomock.Any(), gomock.Any()).Return(true).AnyTimes()

			keeper, ctx := VpoolKeeper(t, pfKeeper)

			ctx = ctx.WithBlockHeader(tmproto.Header{Time: time.Unix(1, 0)})
			// Creation of the pool does NOT trigger a markPriceChanged event
			keeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/*tradeLimitRatio=*/ sdk.OneDec(),
				/*quoteAssetReserve=*/ sdk.NewDec(40_000_000),
				/*baseAssetReserve=*/ sdk.NewDec(1_000),
				/*fluctuationLimitratio=*/ sdk.OneDec(),
				/*maxSpread=*/ sdk.OneDec(),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
			)
			err := keeper.UpdateTWAP(ctx, common.PairBTCStable.String())
			require.NoError(t, err)
			// Make sure price gets initialized correctly when the pool gets created
			pair := common.PairBTCStable
			twap, err := keeper.GetCurrentTWAP(ctx, pair)
			require.NoError(t, err)
			require.EqualValues(t, initialTWAP, twap.Price)
			for i, p := range tc.positionUpdates {
				// update the position and trigger TWAP recalculation
				ctx = ctx.WithBlockHeader(tmproto.Header{Time: p.blockTs})
				if p.baseAsset.IsNil() {
					_, err = keeper.SwapQuoteForBase(ctx, common.PairBTCStable, p.direction, p.quoteAsset, sdk.NewDec(0), true)
				} else {
					_, err = keeper.SwapBaseForQuote(ctx, common.PairBTCStable, p.direction, p.baseAsset, sdk.NewDec(0), true)
				}
				require.NoError(t, err)
				markPriceEvt := getMarkPriceEvent(tc.expectedMarkPrices[i], ctx.BlockHeader().Time)
				testutilevents.RequireContainsTypedEvent(t, ctx, markPriceEvt)
				err = keeper.UpdateTWAP(ctx, common.PairBTCStable.String())
				require.NoError(t, err)
				twap, err := keeper.GetCurrentTWAP(ctx, pair)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedTWAPs[i], twap.Price)
			}
		})
	}
}

func getMarkPriceEvent(price sdk.Dec, ts time.Time) proto.Message {
	return &types.MarkPriceChanged{
		Pair:      common.PairBTCStable.String(),
		Price:     price,
		Timestamp: ts,
	}
}
