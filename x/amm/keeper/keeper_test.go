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

	return Keeper{storeKey: storeKey}
}

func TestSwapInput(t *testing.T) {

}
