package testutil

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/keeper"
	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
)

func NewDexKeeper(
	t testing.TB,
	storeKey storetypes.StoreKey,
	stateStore storetypes.CommitMultiStore,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	cdc *codec.ProtoCodec,
) keeper.Keeper {
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	paramsSubspace := paramstypes.NewSubspace(
		cdc,
		types.Amino,
		storeKey,
		storetypes.NewMemoryStoreKey(types.MemStoreKey),
		"DexParams",
	)

	return *keeper.NewKeeper(
		cdc,
		storeKey,
		paramsSubspace,
		accountKeeper,
		bankKeeper,
	)
}
