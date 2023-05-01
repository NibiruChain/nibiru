package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

func PerpAmmKeeper(t *testing.T, oracleKeeper types.OracleKeeper) (
	perpammKeeper Keeper, ctx sdk.Context,
) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	perpammKeeper = NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		storeKey, oracleKeeper,
	)
	ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return perpammKeeper, ctx
}

// holds mocks for interfaces defined in perp/amm/types/expected_keepers.go
type mockedDependencies struct {
	mockOracleKeeper  *mock.MockOracleKeeper
	mockAccountKeeper *mock.MockAccountKeeper
}

func getKeeper(t *testing.T) (Keeper, mockedDependencies, sdk.Context) {
	db := tmdb.NewMemDB()
	commitMultiStore := store.NewCommitMultiStore(db)
	// Mount the KV store with the x/perp store key
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	commitMultiStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	// Mount Transient store
	transientStoreKey := sdk.NewTransientStoreKey("transient" + types.StoreKey)
	commitMultiStore.MountStoreWithDB(transientStoreKey, sdk.StoreTypeTransient, nil)
	// Mount Memory store
	memStoreKey := storetypes.NewMemoryStoreKey("mem" + types.StoreKey)
	commitMultiStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, commitMultiStore.LoadLatestVersion())

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	ctrl := gomock.NewController(t)
	mockedAccountKeeper := mock.NewMockAccountKeeper(ctrl)
	mockedOracleKeeper := mock.NewMockOracleKeeper(ctrl)

	mockedAccountKeeper.
		EXPECT().GetModuleAddress(types.ModuleName).
		Return(authtypes.NewModuleAddress(types.ModuleName)).AnyTimes()

	k := NewKeeper(
		protoCodec,
		storeKey,
		mockedOracleKeeper,
	)

	ctx := sdk.NewContext(commitMultiStore, tmproto.Header{}, false, log.NewNopLogger())

	return k, mockedDependencies{
		mockOracleKeeper:  mockedOracleKeeper,
		mockAccountKeeper: mockedAccountKeeper,
	}, ctx
}
