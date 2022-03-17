package keeper

import (
	ammv1 "github.com/MatrixDao/matrix/api/amm"
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmdb "github.com/tendermint/tm-db"

	ammtypes "github.com/MatrixDao/matrix/x/amm/types"
)

func AmmKeeper(t *testing.T) Keeper {
	storeKey := sdktypes.NewKVStoreKey(ammtypes.StoreKey)

	db := tmdb.NewMemDB()

	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, types.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	return NewKeeper(storeKey)
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
