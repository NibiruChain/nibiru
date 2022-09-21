package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestGetReserveSnapshotFailsIfNotSnapshotSavedBefore(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	_, err := vpoolKeeper.GetLatestReserveSnapshot(ctx, common.Pair_BTC_NUSD)
	require.Error(t, err, types.ErrNoLastSnapshotSaved)
}

func TestGetReserveSnapshotMultiplePairs(t *testing.T) {
	genesisTime := time.Now()
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	ctx = ctx.WithBlockHeight(1).WithBlockTime(genesisTime)
	snapshot := types.NewReserveSnapshot(
		common.Pair_BTC_NUSD,
		sdk.OneDec(),
		sdk.OneDec(),
		ctx.BlockTime(),
		ctx.BlockHeight(),
	)
	vpoolKeeper.SaveSnapshot(ctx, snapshot)

	snapshot = types.NewReserveSnapshot(
		common.Pair_ETH_NUSD,
		sdk.NewDec(2),
		sdk.NewDec(2),
		ctx.BlockTime(),
		ctx.BlockHeight(),
	)
	vpoolKeeper.SaveSnapshot(
		ctx,
		snapshot,
	)
	ctx = ctx.WithBlockHeight(2).WithBlockTime(genesisTime.Add(5 * time.Second))

	snapshot, err := vpoolKeeper.GetLatestReserveSnapshot(ctx, common.Pair_BTC_NUSD)
	require.NoError(t, err)
	require.Equal(t,
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.OneDec(),
			sdk.OneDec(),
			genesisTime,
			1,
		),
		snapshot,
	)
}

func TestSaveSnapshot(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	snapshot := types.NewReserveSnapshot(common.Pair_BTC_NUSD, sdk.OneDec(), sdk.OneDec(), ctx.BlockTime(), ctx.BlockHeight())
	vpoolKeeper.SaveSnapshot(ctx, snapshot)

	snapshot, err := vpoolKeeper.GetLatestReserveSnapshot(ctx, common.Pair_BTC_NUSD)
	require.NoError(t, err)
	require.Equal(t,
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.OneDec(),
			sdk.OneDec(),
			ctx.BlockTime(),
			1,
		),
		snapshot,
	)
}

func TestGetSnapshot(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	t.Log("Save snapshot 1")
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())
	snapshot := types.NewReserveSnapshot(
		common.Pair_BTC_NUSD,
		sdk.OneDec(),
		sdk.OneDec(),
		ctx.BlockTime(),
		ctx.BlockHeight(),
	)
	vpoolKeeper.SaveSnapshot(ctx, snapshot)

	t.Log("Check snapshot 1")
	snapshot, err := vpoolKeeper.GetSnapshot(ctx, common.Pair_BTC_NUSD, 1)
	require.NoError(t, err)
	require.Equal(t,
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.OneDec(),
			sdk.OneDec(),
			ctx.BlockTime(),
			1,
		),
		snapshot,
	)

	t.Log("Save snapshot 2")
	ctx = ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(5 * time.Second))
	snapshot = types.NewReserveSnapshot(
		common.Pair_BTC_NUSD,
		sdk.NewDec(2),
		sdk.NewDec(2),
		ctx.BlockTime(),
		ctx.BlockHeight(),
	)
	vpoolKeeper.SaveSnapshot(
		ctx,
		snapshot,
	)

	t.Log("Fetch snapshot 2")
	snapshot, err = vpoolKeeper.GetSnapshot(ctx, common.Pair_BTC_NUSD, 2)
	require.NoError(t, err)
	require.Equal(t,
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.NewDec(2),
			sdk.NewDec(2),
			ctx.BlockTime(),
			2,
		),
		snapshot,
	)
}

func TestGetAllSnapshots(t *testing.T) {
	vpoolKeeper, ctx := VpoolKeeper(t,
		mock.NewMockPricefeedKeeper(gomock.NewController(t)),
	)

	snapshots := []types.ReserveSnapshot{
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.NewDec(1_000_000),
			sdk.NewDec(60_000_000_000),
			time.UnixMilli(123456),
			1,
		),
		types.NewReserveSnapshot(
			common.Pair_BTC_NUSD,
			sdk.NewDec(2_000_000),
			sdk.NewDec(50_000_000_000),
			time.UnixMilli(223456),
			2,
		),
	}

	for _, snapshot := range snapshots {
		vpoolKeeper.SaveSnapshot(ctx, snapshot)
	}

	savedSnapshots := vpoolKeeper.GetAllSnapshots(ctx)
	require.Len(t, savedSnapshots, 2)

	for _, snapshot := range snapshots {
		require.Contains(t, savedSnapshots, snapshot)
	}
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
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(2),
			twapCalcOption:    types.TwapCalcOption_SPOT,
			expectedPrice:     sdk.NewDec(20_000),
		},
		{
			name:              "quote asset swap add to pool calc",
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_ADD_TO_POOL,
			assetAmount:       sdk.NewDec(3_000),
			expectedPrice:     sdk.NewDec(500),
		},
		{
			name:              "quote asset swap remove from pool calc",
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_QUOTE_ASSET_SWAP,
			direction:         types.Direction_REMOVE_FROM_POOL,
			assetAmount:       sdk.NewDec(1_500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap add to pool calc",
			pair:              common.Pair_BTC_NUSD,
			quoteAssetReserve: sdk.NewDec(3_000),
			baseAssetReserve:  sdk.NewDec(1_000),
			twapCalcOption:    types.TwapCalcOption_BASE_ASSET_SWAP,
			direction:         types.Direction_ADD_TO_POOL,
			assetAmount:       sdk.NewDec(500),
			expectedPrice:     sdk.NewDec(1_000),
		},
		{
			name:              "base asset swap remove from pool calc",
			pair:              common.Pair_BTC_NUSD,
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
			snapshot := types.NewReserveSnapshot(
				tc.pair,
				tc.baseAssetReserve,
				tc.quoteAssetReserve,
				time.Now(),
				1,
			)

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
