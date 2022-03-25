package testutil

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
)

func NewBankKeeper(
	t testing.TB,
	stateStore storetypes.CommitMultiStore,
	accountKeeper authkeeper.AccountKeeper,
	cdc *codec.ProtoCodec,
) bankkeeper.Keeper {
	storeKey := storetypes.NewKVStoreKey("bank")
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	paramsSubspace := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		storetypes.NewMemoryStoreKey("mem_bank"),
		"AccountParams",
	)

	return bankkeeper.NewBaseKeeper(
		cdc,
		storeKey,
		accountKeeper,
		paramsSubspace,
		map[string]bool{},
	)
}
