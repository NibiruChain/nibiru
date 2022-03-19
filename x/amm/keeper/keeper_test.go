package keeper

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/orm/model/ormdb"
	"github.com/cosmos/cosmos-sdk/orm/model/ormtable"
	"github.com/cosmos/cosmos-sdk/orm/testing/ormtest"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	ammv1 "github.com/MatrixDao/matrix/api/amm"
	ammtypes "github.com/MatrixDao/matrix/x/amm/types"
)

const UsdmPair = "BTC:USDM"

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
		pair        string
		direction   ammv1.Direction
		quoteAmount sdktypes.Int
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			ammv1.Direction_ADD_TO_AMM,
			sdktypes.NewInt(10),
			ammtypes.ErrPairNotSupported,
		},
		{
			"quote input bigger than reserve ratio",
			UsdmPair,
			ammv1.Direction_REMOVE_FROM_AMM,
			sdktypes.NewInt(10_000_000),
			ammtypes.ErrOvertradingLimit,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper := AmmKeeper(t)

			err := keeper.CreatePool(
				context.Background(),
				UsdmPair,
				sdktypes.NewInt(900_000),    // 0.9 ratio
				sdktypes.NewInt(10_000_000), // 10
				sdktypes.NewInt(5_000_000),  // 5
			)
			require.NoError(t, err)

			_, err = keeper.SwapInput(tc.pair, tc.direction, tc.quoteAmount)
			require.EqualError(t, err, tc.error.Error())
		})
	}
}

func TestSwapInput_HappyPath(t *testing.T) {
	tests := []struct {
		name        string
		direction   ammv1.Direction
		quoteAmount sdktypes.Int
		resp        sdktypes.Int
	}{
		{
			"quote amount == 0",
			ammv1.Direction_ADD_TO_AMM,
			sdktypes.NewInt(0),
			sdktypes.ZeroInt(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper := AmmKeeper(t)

			err := keeper.CreatePool(
				context.Background(),
				UsdmPair,
				sdktypes.NewInt(900_000),    // 0.9 ratio
				sdktypes.NewInt(10_000_000), // 10 tokens
				sdktypes.NewInt(5_000_000),  // 5 tokens
			)
			require.NoError(t, err)

			res, err := keeper.SwapInput(UsdmPair, tc.direction, tc.quoteAmount)
			require.NoError(t, err)
			require.Equal(t, res, tc.resp)
		})
	}
}

func TestCreatePool(t *testing.T) {
	ammKeeper := AmmKeeper(t)

	err := ammKeeper.CreatePool(
		context.Background(),
		UsdmPair,
		sdktypes.NewInt(900_000),    // 0.9 ratio
		sdktypes.NewInt(10_000_000), // 10 tokens
		sdktypes.NewInt(5_000_000),  // 5 tokens
	)
	require.NoError(t, err)

	exists := ammKeeper.ExistsPool(context.Background(), UsdmPair)
	require.True(t, exists)

	notExist := ammKeeper.ExistsPool(context.Background(), "BTC:OTHER")
	require.False(t, notExist)
}
