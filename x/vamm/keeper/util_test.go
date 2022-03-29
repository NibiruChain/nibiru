package keeper

import (
	ammtypes "github.com/MatrixDao/matrix/x/vamm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	types2 "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
	"testing"
)

const UsdmPair = "BTC:USDM"

func AmmKeeper(t *testing.T) (Keeper, sdktypes.Context) {
	storeKey := sdktypes.NewKVStoreKey(ammtypes.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdktypes.StoreTypeIAVL, db)

	require.NoError(t, stateStore.LoadLatestVersion())

	registry := types2.NewInterfaceRegistry()

	k := NewKeeper(
		codec.NewProtoCodec(registry),
		storeKey,
	)

	ctx := sdktypes.NewContext(stateStore, tmproto.Header{}, false, nil)

	return k, ctx
}

func getSamplePool() *ammtypes.Pool {
	ratioLimit, _ := sdktypes.NewDecFromStr("0.9")
	fluctuationLimit, _ := sdktypes.NewDecFromStr("0.1")

	pool := ammtypes.NewPool(
		UsdmPair,
		ratioLimit,
		sdktypes.NewInt(10_000_000),
		sdktypes.NewInt(5_000_000),
		fluctuationLimit,
	)

	return pool
}
