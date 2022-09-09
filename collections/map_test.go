package collections_test

import (
	"github.com/NibiruChain/nibiru/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/mock"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
	"testing"
)

func deps() (sdk.StoreKey, sdk.Context, codec.BinaryCodec) {
	sk := sdk.NewKVStoreKey("mock")
	dbm := db.NewMemDB()
	ms := mock.NewCommitMultiStore()
	ms.MountStoreWithDB(sk, types.StoreTypeMemory, dbm)
	return sk, sdk.Context{}.WithMultiStore(ms).WithGasMeter(sdk.NewGasMeter(1_000_000_000)), codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func TestMap(t *testing.T) {
	sk, ctx, cdc := deps()
	m := collections.NewMap[collections.StringKey, stakingtypes.MsgBeginRedelegate, *stakingtypes.MsgBeginRedelegate](cdc, sk, 0)

	key := collections.StringKey("hi")
	expected := stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    "me",
		ValidatorSrcAddress: "you",
		ValidatorDstAddress: "him",
		Amount:              sdk.NewInt64Coin("nibi", 10),
	}

	// test insert and get
	m.Insert(ctx, key, expected)
	got, err := m.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	// test delete and get error
	err = m.Delete(ctx, key)
	require.NoError(t, err)
	_, err = m.Get(ctx, key)
	require.ErrorIs(t, err, collections.ErrNotFound)

	// test delete errors not exist
	err = m.Delete(ctx, key)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
