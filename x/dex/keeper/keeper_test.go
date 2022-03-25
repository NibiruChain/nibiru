package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/testutil/nullify"
	"github.com/MatrixDao/matrix/x/dex/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
)

func TestGetNextPoolNumber(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, cdc := testkeeper.NewDexKeeper(t, storeKey)

	// Write to store manually
	bz := cdc.MustMarshal(&gogotypes.UInt64Value{Value: 100})
	ctx.KVStore(storeKey).Set(types.KeyNextGlobalPoolNumber, bz)

	// Read
	poolNumber := k.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 100)
}

func TestSetNextPoolNumber(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)

	// Write to store
	k.SetNextPoolNumber(ctx, 150)

	// Read from store
	poolNumber := k.GetNextPoolNumber(ctx)

	require.EqualValues(t, poolNumber, 150)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)

	// Write a pool number
	k.SetNextPoolNumber(ctx, 200)

	// Get next and increment should return the current pool number
	poolNumber := k.GetNextPoolNumberAndIncrement(ctx)
	require.EqualValues(t, poolNumber, 200)

	// Check that the previous call incremented the number
	poolNumber = k.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 201)
}

func TestSetAndFetchPool(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.ModuleName)
	k, ctx, _ := testkeeper.NewDexKeeper(t, storeKey)

	pool := types.Pool{
		Id: 150,
		PoolParams: types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
		},
		PoolAssets: []types.PoolAsset{
			types.PoolAsset{
				Token: sdk.NewCoin("token", sdk.NewInt(100)),
			},
			types.PoolAsset{
				Token: sdk.NewCoin("token", sdk.NewInt(100)),
			},
		},
	}

	err := k.SetPool(ctx, pool)
	require.NoError(t, err)

	retrievedPool, err := k.FetchPool(ctx, 150)
	require.NoError(t, err)

	nullify.Fill(pool)
	nullify.Fill(retrievedPool)

	require.Equal(t, pool, retrievedPool)
}
