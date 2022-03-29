package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/MatrixDao/matrix/x/testutil/nullify"
	"github.com/MatrixDao/matrix/x/testutil/sample"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestGetAndSetNextPoolNumber(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	// Write to store
	app.DexKeeper.SetNextPoolNumber(ctx, 150)

	// Read from store
	poolNumber := app.DexKeeper.GetNextPoolNumber(ctx)

	require.EqualValues(t, poolNumber, 150)
}

func TestGetNextPoolNumberAndIncrement(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	// Write a pool number
	app.DexKeeper.SetNextPoolNumber(ctx, 200)

	// Get next and increment should return the current pool number
	poolNumber := app.DexKeeper.GetNextPoolNumberAndIncrement(ctx)
	require.EqualValues(t, poolNumber, 200)

	// Check that the previous call incremented the number
	poolNumber = app.DexKeeper.GetNextPoolNumber(ctx)
	require.EqualValues(t, poolNumber, 201)
}

func TestSetAndFetchPool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	pool := types.Pool{
		Id: 150,
		PoolParams: types.PoolParams{
			SwapFee: sdk.NewDecWithPrec(3, 2),
			ExitFee: sdk.NewDecWithPrec(3, 2),
		},
		PoolAssets: []types.PoolAsset{
			types.PoolAsset{
				Token: sdk.NewCoin("validatortoken", sdk.NewInt(1000)),
			},
			types.PoolAsset{
				Token: sdk.NewCoin("stake", sdk.NewInt(1000)),
			},
		},
	}

	err := app.DexKeeper.SetPool(ctx, pool)
	require.NoError(t, err)

	retrievedPool, err := app.DexKeeper.FetchPool(ctx, 150)
	require.NoError(t, err)

	nullify.Fill(pool)
	nullify.Fill(retrievedPool)

	require.Equal(t, pool, retrievedPool)
}

func TestNewPool(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()
	app.DexKeeper.SetNextPoolNumber(ctx, 1)

	userAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	coins := sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000)),
	)

	err := simapp.FundAccount(app.BankKeeper, ctx, userAddr, coins)
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uosmo", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.DexKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.NoError(t, err)

	retrievedPool, err := app.DexKeeper.FetchPool(ctx, poolId)
	require.NoError(t, err)
	require.Equal(t, poolAssets, retrievedPool.PoolAssets)
	require.Equal(t, poolParams, retrievedPool.PoolParams)

	liquidity := app.DexKeeper.GetTotalLiquidity(ctx)
	require.Equal(t, sdk.NewCoins(
		sdk.NewCoin("uatom", sdk.NewInt(1000)),
		sdk.NewCoin("uosmo", sdk.NewInt(1000))),
		liquidity)
}

func TestNewPoolTooLittleAssets(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()
	userAddr, err := sdk.AccAddressFromBech32(sample.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.DexKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooFewPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestNewPoolTooManyAssets(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()
	userAddr, err := sdk.AccAddressFromBech32(sample.AccAddress().String())
	require.NoError(t, err)

	poolParams := types.PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []types.PoolAsset{
		{
			Token: sdk.NewCoin("uatom1", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom2", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom3", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom4", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom5", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom6", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom7", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom8", sdk.NewInt(1000)),
		},
		{
			Token: sdk.NewCoin("uatom9", sdk.NewInt(1000)),
		},
	}

	poolId, err := app.DexKeeper.NewPool(ctx, userAddr, poolParams, poolAssets)
	require.ErrorIs(t, err, types.ErrTooManyPoolAssets)
	require.Equal(t, uint64(0), poolId)
}

func TestMintPoolShareToAccount(t *testing.T) {
	app, ctx := testutil.NewMatrixApp()

	userAddr, err := sdk.AccAddressFromBech32(sample.AccAddress().String())
	require.NoError(t, err)

	err = app.DexKeeper.MintPoolShareToAccount(ctx, 1, userAddr, sdk.NewIntWithDecimal(100, 18))
	require.NoError(t, err)

	coin := app.BankKeeper.GetBalance(ctx, userAddr, "matrix/pool/1")
	require.Equal(t, sdk.NewIntWithDecimal(100, 18), coin.Amount)
}
