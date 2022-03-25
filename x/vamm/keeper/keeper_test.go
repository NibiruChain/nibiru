package keeper

import (
	"github.com/tendermint/tendermint/types/time"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	types2 "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	ammtypes "github.com/MatrixDao/matrix/x/vamm/types"
)

const UsdmPair = "BTC:USDM"

func AmmKeeper(t *testing.T) (Keeper, sdktypes.Context) {
	storeKey := sdktypes.NewKVStoreKey(ammtypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdktypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, sdktypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := types2.NewInterfaceRegistry()

	k := NewKeeper(
		codec.NewProtoCodec(registry),
		storeKey,
	)

	ctx := sdktypes.NewContext(stateStore, tmproto.Header{}, false, nil)

	return k, ctx
}

func TestSwapInput_Errors(t *testing.T) {
	tests := []struct {
		name        string
		pair        string
		direction   ammtypes.Direction
		quoteAmount sdktypes.Int
		baseLimit   sdktypes.Int
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(10),
			sdktypes.NewInt(10),
			ammtypes.ErrPairNotSupported,
		},
		{
			"quote input bigger than reserve ratio",
			UsdmPair,
			ammtypes.Direction_REMOVE_FROM_AMM,
			sdktypes.NewInt(10_000_000),
			sdktypes.NewInt(10),
			ammtypes.ErrOvertradingLimit,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				UsdmPair,
				sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdktypes.NewInt(10_000_000),       // 10
				sdktypes.NewInt(5_000_000),        // 5
			)
			require.NoError(t, err)

			_, err = keeper.SwapInput(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.EqualError(t, err, tc.error.Error())
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name                 string
		direction            ammtypes.Direction
		quoteAmount          sdktypes.Int
		baseLimit            sdktypes.Int
		expectedQuoteReserve sdktypes.Int
		expectedBaseReserve  sdktypes.Int
		resp                 sdktypes.Int
	}{
		{
			"quote amount == 0",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(0),
			sdktypes.NewInt(10),
			sdktypes.NewInt(10_000_000),
			sdktypes.NewInt(5_000_000),
			sdktypes.ZeroInt(),
		},
		{
			"normal swap add",
			ammtypes.Direction_ADD_TO_AMM,
			sdktypes.NewInt(1_000_000),
			sdktypes.NewInt(454_500),
			sdktypes.NewInt(11_000_000),
			sdktypes.NewInt(4_545_456),
			sdktypes.NewInt(454_544),
		},
		{
			"normal swap remove",
			ammtypes.Direction_REMOVE_FROM_AMM,
			sdktypes.NewInt(1_000_000),
			sdktypes.NewInt(555_560),
			sdktypes.NewInt(9_000_000),
			sdktypes.NewInt(5_555_556),
			sdktypes.NewInt(555_556),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := AmmKeeper(t)

			err := keeper.CreatePool(
				ctx,
				UsdmPair,
				sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdktypes.NewInt(10_000_000),       // 10 tokens
				sdktypes.NewInt(5_000_000),        // 5 tokens
			)
			require.NoError(t, err)

			res, err := keeper.SwapInput(
				ctx,
				UsdmPair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.NoError(t, err)
			require.Equal(t, res, tc.resp)

			pool, err := keeper.getPool(ctx, UsdmPair)
			quoteAmount, err := pool.GetPoolQuoteAssetReserveAsInt()
			require.NoError(t, err)
			require.Equal(t, tc.expectedQuoteReserve, quoteAmount)

			baseAmount, err := pool.GetPoolBaseAssetReserveAsInt()
			require.NoError(t, err)
			require.Equal(t, tc.expectedBaseReserve, baseAmount)
		})
	}
}

func TestCreatePool(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	err := ammKeeper.CreatePool(
		ctx,
		UsdmPair,
		sdktypes.MustNewDecFromStr("0.9"), // 0.9 ratio
		sdktypes.NewInt(10_000_000),       // 10 tokens
		sdktypes.NewInt(5_000_000),        // 5 tokens
	)
	require.NoError(t, err)

	exists := ammKeeper.ExistsPool(ctx, UsdmPair)
	require.True(t, exists)

	notExist := ammKeeper.ExistsPool(ctx, "BTC:OTHER")
	require.False(t, notExist)
}

func TestKeeper_AddReserveSnapshot_ErrorNoLastSnapshot(t *testing.T) {
	ammKeeper, ctx := AmmKeeper(t)

	_, err := ammKeeper.GetLastReserveSnapshot(ctx, UsdmPair)
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)
}

func TestKeeper_AddReserveSnapshot(t *testing.T) {
	expectedTime := time.Now()
	expectedBlockHeight := 123
	ammKeeper, ctx := AmmKeeper(t)
	ctx = ctx.WithBlockHeight(int64(expectedBlockHeight))
	ctx = ctx.WithBlockTime(expectedTime)

	_, err := ammKeeper.GetLastReserveSnapshot(ctx, UsdmPair)
	require.Error(t, err, ammtypes.ErrNoLastSnapshotSaved)

	ratioLimit, err := sdktypes.NewDecFromStr("0.9")
	pool := ammtypes.NewPool(
		UsdmPair,
		ratioLimit,
		sdktypes.NewInt(10_000_000),
		sdktypes.NewInt(5_000_000),
	)

	err = ammKeeper.TakeReserveSnapshot(ctx, pool)
	require.NoError(t, err)

	snapshot, err := ammKeeper.GetLastReserveSnapshot(ctx, UsdmPair)
	require.NoError(t, err)

	require.Equal(t, ammtypes.ReserveSnapshot{
		QuoteAssetReserve: pool.QuoteAssetReserve,
		BaseAssetReserve:  pool.BaseAssetReserve,
		Timestamp:         expectedTime.Unix(),
		BlockNumber:       int64(expectedBlockHeight),
	}, snapshot)
}
