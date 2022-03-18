package keeper

import (
	"context"
	ammv1 "github.com/MatrixDao/matrix/api/amm"
	"github.com/cosmos/cosmos-sdk/orm/model/ormdb"
	"github.com/cosmos/cosmos-sdk/orm/model/ormtable"
	"github.com/cosmos/cosmos-sdk/orm/testing/ormtest"
	"testing"

	ammtypes "github.com/MatrixDao/matrix/x/amm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func AmmKeeper(t *testing.T) Keeper {
	db := ormtest.NewMemoryBackend()

	moduleDB, err := ormdb.NewModuleDB(PoolSchema, ormdb.ModuleDBOptions{
		GetBackend: func(ctx context.Context) (ormtable.Backend, error) {
			return db, nil
		},
		GetReadBackend: func(ctx context.Context) (ormtable.ReadBackend, error) {
			return db, nil
		},
	})
	require.NoError(t, err)

	ammStore, err := ammv1.NewAmmStore(moduleDB)
	require.NoError(t, err)

	return NewKeeper(ammStore)
}

func TestSwapInput_Errors(t *testing.T) {
	tests := []struct {
		name        string
		direction   ammv1.Direction
		quoteAmount sdktypes.Coin
		error       error
	}{
		{
			"amount not USDM",
			ammv1.Direction_ADD_TO_AMM,
			sdktypes.NewCoin("uusdt", sdktypes.NewInt(10)),
			ammtypes.ErrStableNotSupported,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper := AmmKeeper(t)
			_, err := keeper.SwapInput(tc.direction, tc.quoteAmount)
			require.EqualError(t, err, tc.error.Error())
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name        string
		direction   ammv1.Direction
		quoteAmount sdktypes.Coin
		resp        sdktypes.Int
	}{
		{
			"quote amount == 0",
			ammv1.Direction_ADD_TO_AMM,
			sdktypes.NewCoin("uusdm", sdktypes.NewInt(0)),
			sdktypes.ZeroInt(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper := AmmKeeper(t)
			res, err := keeper.SwapInput(tc.direction, tc.quoteAmount)
			require.NoError(t, err)
			require.Equal(t, res, tc.resp)
		})
	}
}

func TestCreatePool(t *testing.T) {
	ammKeeper := AmmKeeper(t)

	err := ammKeeper.CreatePool(context.Background(), "BTC:USDM")
	require.NoError(t, err)

	pool, err := ammKeeper.GetPool(context.Background(), "BTC:USDM")
	require.NoError(t, err)

	require.Equal(t, pool.Pair, "BTC:USDM")
}
