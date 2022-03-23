package keeper

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

func DexKeeper(t testing.TB) (*keeper.Keeper, sdk.Context, *codec.ProtoCodec, storetypes.StoreKey) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	// stateStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := paramstypes.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"DexParams",
	)
	accountKeeper, _ := AccountKeeper(t)
	bankKeeper, _ := BankKeeper(t, accountKeeper)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		paramsSubspace,
		accountKeeper,
		bankKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return k, ctx, cdc, storeKey
}
