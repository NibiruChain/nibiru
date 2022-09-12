package collections_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wellknown "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/collections"
	"github.com/NibiruChain/nibiru/collections/keys"
)

func deps() (sdk.StoreKey, sdk.Context, codec.BinaryCodec) {
	sk := sdk.NewKVStoreKey("mock")
	dbm := db.NewMemDB()
	ms := store.NewCommitMultiStore(dbm)
	ms.MountStoreWithDB(sk, types.StoreTypeIAVL, dbm)
	if err := ms.LoadLatestVersion(); err != nil {
		panic(err)
	}
	return sk, sdk.Context{}.WithMultiStore(ms).WithGasMeter(sdk.NewGasMeter(1_000_000_000)), codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func obj(o string) wellknown.BytesValue {
	return wellknown.BytesValue{Value: []byte(o)}
}

func kv(o string) collections.KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue] {
	return collections.KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue]{
		Key:   keys.StringKey(o),
		Value: wellknown.BytesValue{Value: []byte(o)},
	}
}

func TestMap(t *testing.T) {
	sk, ctx, cdc := deps()
	m := collections.NewMap[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue](cdc, sk, 0)

	key := keys.String("id")
	expected := obj("test")

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

func TestMap_Iterate(t *testing.T) {
	sk, ctx, cdc := deps()
	m := collections.NewMap[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue](cdc, sk, 0)

	objs := []collections.KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue]{kv("a"), kv("aa"), kv("b"), kv("bb")}

	m.Insert(ctx, "a", obj("a"))
	m.Insert(ctx, "aa", obj("aa"))
	m.Insert(ctx, "b", obj("b"))
	m.Insert(ctx, "bb", obj("bb"))

	// test iteration ascending
	iter := m.Iterate(ctx, keys.None[keys.StringKey](), keys.None[keys.StringKey](), keys.OrderAscending)
	defer iter.Close()
	for i, o := range iter.All() {
		require.Equal(t, objs[i], o)
	}

	// test iteration descending
	dIter := m.Iterate(ctx, keys.None[keys.StringKey](), keys.None[keys.StringKey](), keys.OrderDescending)
	defer dIter.Close()
	for i, o := range iter.All() {
		require.Equal(t, objs[len(objs)-1-i], o)
	}

	// test all keys

	// test all values
}
