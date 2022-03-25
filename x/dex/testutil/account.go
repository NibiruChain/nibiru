package testutil

import (
	"testing"

	"github.com/MatrixDao/matrix/x/dex/types"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
)

func NewAccountKeeper(
	t testing.TB,
	stateStore storetypes.CommitMultiStore,
	cdc *codec.ProtoCodec,
) keeper.AccountKeeper {
	storeKey := storetypes.NewKVStoreKey("account")
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	paramsSubspace := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		storetypes.NewMemoryStoreKey("mem_account"),
		"AccountParams",
	)

	return keeper.NewAccountKeeper(
		cdc,
		storeKey,
		paramsSubspace,
		authtypes.ProtoBaseAccount,
		map[string][]string{
			types.ModuleName: nil,
		},
	)
}
