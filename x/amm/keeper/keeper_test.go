package keeper

import (
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
		direction   ammtypes.Direction
		quoteAmount sdktypes.Coin
	}{
		{
			"amount not USDM",
			ammtypes.ADD_TO_AMM,
			sdktypes.NewCoin("USDM", sdktypes.NewInt(10)),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			keeper := AmmKeeper(t)
			err := keeper.SwapInput(tc.direction, tc.quoteAmount)
			require.EqualError(t, err, "")
		})
	}
}
