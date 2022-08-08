package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestKeeper_saveOrGetReserveSnapshotFailsIfNotSnapshotSavedBefore(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	pool := getSamplePool()

	err := vpoolKeeper.updateSnapshot(ctx, pool.Pair, pool.QuoteAssetReserve, pool.BaseAssetReserve)
	require.Error(t, err, types.ErrNoLastSnapshotSaved)

	_, _, err = vpoolKeeper.getLatestReserveSnapshot(ctx, common.PairBTCStable)
	require.Error(t, err, types.ErrNoLastSnapshotSaved)
}

func TestSaveSnapshot(t *testing.T) {
	expectedTime := tmtime.Now()
	expectedBlockHeight := int64(123)
	pool := getSamplePool()

	expectedSnapshot := types.ReserveSnapshot{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
		TimestampMs:       expectedTime.UnixMilli(),
		BlockNumber:       expectedBlockHeight,
	}

	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	ctx = ctx.WithBlockHeight(expectedBlockHeight).WithBlockTime(expectedTime)
	vpoolKeeper.saveSnapshot(ctx, pool.Pair, 0, pool.QuoteAssetReserve, pool.BaseAssetReserve)
	vpoolKeeper.saveSnapshotCounter(ctx, pool.Pair, 0)

	snapshot, counter, err := vpoolKeeper.getLatestReserveSnapshot(ctx, common.PairBTCStable)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshot, snapshot)
	require.Equal(t, uint64(0), counter)
}

func TestGetSnapshot(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	expectedHeight := int64(123)
	expectedTime := tmtime.Now()

	ctx = ctx.WithBlockHeight(expectedHeight).WithBlockTime(expectedTime)

	pool := getSamplePool()

	firstSnapshot := types.ReserveSnapshot{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
		TimestampMs:       expectedTime.UnixMilli(),
		BlockNumber:       expectedHeight,
	}

	t.Log("Save snapshot 0")
	vpoolKeeper.saveSnapshot(
		ctx,
		pool.Pair,
		0,
		pool.QuoteAssetReserve,
		pool.BaseAssetReserve,
	)
	vpoolKeeper.saveSnapshotCounter(ctx, pool.Pair, 0)

	t.Log("Check snapshot 0")
	oldSnapshot, counter, err := vpoolKeeper.getLatestReserveSnapshot(ctx, pool.Pair)
	require.NoError(t, err)
	require.Equal(t, firstSnapshot, oldSnapshot)
	require.Equal(t, uint64(0), counter)

	t.Log("We save another different snapshot")
	differentSnapshot := firstSnapshot
	differentSnapshot.BaseAssetReserve = sdk.NewDec(12_341_234)
	differentSnapshot.BlockNumber = expectedHeight + 1
	differentSnapshot.TimestampMs = expectedTime.Add(time.Second).UnixMilli()
	pool.BaseAssetReserve = differentSnapshot.BaseAssetReserve
	ctx = ctx.WithBlockHeight(expectedHeight + 1).WithBlockTime(expectedTime.Add(time.Second))
	vpoolKeeper.saveSnapshot(
		ctx,
		pool.Pair,
		1,
		pool.QuoteAssetReserve,
		pool.BaseAssetReserve,
	)

	t.Log("Fetch snapshot 1")
	newSnapshot, err := vpoolKeeper.getSnapshot(ctx, pool.Pair, 1)
	require.NoError(t, err)
	require.Equal(t, differentSnapshot, newSnapshot)
	require.NotEqual(t, differentSnapshot, oldSnapshot)
}

func TestGetSnapshotPrice(t *testing.T) {
	tests := []struct {
		name              string
		pair              common.AssetPair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		twapCalcOption    types.TwapCalcOption
		direction         types.Direction
		assetAmount       sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "spot price calc",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(2),
			twapCalcOption:    types.TwapCalcOption_SPOT,
			expectedPrice:     sdk.NewDec(20_000),
		},
		{
			name:              "quote asset swap add to pool calc",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_ADD_TO_POOL,
			assetAmount:       sdk.NewDec(3_000),
			expectedPrice:     sdk.NewDec(500),
		},
		{
			name:              "quote asset swap remove from pool calc",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_REMOVE_FROM_POOL,
			assetAmount:       sdk.NewDec(1_500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap add to pool calc",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:         types.Direction_ADD_TO_POOL,
			assetAmount:       sdk.NewDec(500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap remove from pool calc",
			pair:              common.PairBTCStable,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:         types.Direction_REMOVE_FROM_POOL,
			assetAmount:       sdk.NewDec(500),
			expectedPrice:     sdk.NewDec(3_000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			snapshot := types.ReserveSnapshot{
				QuoteAssetReserve: tc.quoteAssetReserve,
				BaseAssetReserve:  tc.baseAssetReserve,
			}

			snapshotPriceOpts := snapshotPriceOptions{
				pair:           tc.pair,
				twapCalcOption: tc.twapCalcOption,
				direction:      tc.direction,
				assetAmount:    tc.assetAmount,
			}

			price, err := getPriceWithSnapshot(
				snapshot,
				snapshotPriceOpts,
			)

			require.NoError(t, err)
			require.EqualValuesf(t, tc.expectedPrice, price,
				"expected %s, got %s", tc.expectedPrice.String(), price.String())
		})
	}
}
