package keeper_test

import (
	"testing"

	testkeeper "github.com/MatrixDao/matrix/testutil/keeper"
	"github.com/MatrixDao/matrix/testutil/nullify"
	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetAndSetNextPoolNumber(t *testing.T) {
	k, ctx, _, _ := testkeeper.DexKeeper(t)

	k.SetNextPoolNumber(ctx, 100)
	poolNumber := k.GetNextPoolNumber(ctx)

	require.EqualValues(t, poolNumber, 100)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	k, ctx, _, _ := testkeeper.DexKeeper(t)

	k.SetNextPoolNumber(ctx, 200)

	poolNumber := k.GetNextPoolNumberAndIncrement(ctx)
	require.EqualValues(t, poolNumber, 200)

	poolNumber = k.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 201)
}

func TestSetAndFetchPool(t *testing.T) {
	k, ctx, _, _ := testkeeper.DexKeeper(t)

	pool := types.Pool{
		Id: 150,
		PoolParams: types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
		},
		PoolAssets: []types.PoolAsset{
			types.PoolAsset{
				Token:  sdk.NewCoin("token", sdk.NewInt(100)),
				Weight: sdk.NewInt(50),
			},
			types.PoolAsset{
				Token:  sdk.NewCoin("token", sdk.NewInt(100)),
				Weight: sdk.NewInt(50),
			},
		},
		TotalWeight: sdk.NewInt(100),
	}

	err := k.SetPool(ctx, pool)
	require.NoError(t, err)

	retrievedPool, err := k.FetchPool(ctx, 150)

	nullify.Fill(&pool)
	nullify.Fill(retrievedPool)

	require.Equal(t, pool, *retrievedPool)
}
