package collections

import (
	"testing"

	wellknown "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections/keys"
)

func obj(o string) wellknown.BytesValue {
	return wellknown.BytesValue{Value: []byte(o)}
}

func kv(o string) KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue] {
	return KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue]{
		Key:   keys.StringKey(o),
		Value: wellknown.BytesValue{Value: []byte(o)},
	}
}

func TestUpstreamIterAssertions(t *testing.T) {
	// ugly but asserts upstream behavior
	sk, ctx, _ := deps()
	kv := ctx.KVStore(sk)
	kv.Set([]byte("hi"), []byte{})
	i := kv.Iterator(nil, nil)
	err := i.Close()
	require.NoError(t, err)
	require.NoError(t, i.Close())
}

func TestMap(t *testing.T) {
	sk, ctx, cdc := deps()
	m := NewMap[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue](cdc, sk, 0)

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
	require.ErrorIs(t, err, ErrNotFound)

	// test delete errors not exist
	err = m.Delete(ctx, key)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMapGetOrDefault(t *testing.T) {
	sk, ctx, cdc := deps()
	m := NewMap[keys.StringKey, wellknown.UInt64Value, *wellknown.UInt64Value](cdc, sk, 0)
	assert.EqualValues(t, wellknown.UInt64Value{Value: 123}, m.GetOr(ctx, "foo", wellknown.UInt64Value{Value: 123}))

	m.Insert(ctx, "foo", wellknown.UInt64Value{Value: 456})
	assert.EqualValues(t, wellknown.UInt64Value{Value: 456}, m.GetOr(ctx, "foo", wellknown.UInt64Value{Value: 123}))
}

func TestMapIterate(t *testing.T) {
	sk, ctx, cdc := deps()
	m := NewMap[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue](cdc, sk, 0)

	expectedObjs := []KeyValue[keys.StringKey, wellknown.BytesValue, *wellknown.BytesValue]{
		kv("a"), kv("aa"), kv("b"), kv("bb"),
	}

	m.Insert(ctx, "a", obj("a"))
	m.Insert(ctx, "aa", obj("aa"))
	m.Insert(ctx, "b", obj("b"))
	m.Insert(ctx, "bb", obj("bb"))

	// test iteration ascending
	iter := m.Iterate(ctx, keys.NewRange[keys.StringKey]())
	defer iter.Close()
	for i, o := range iter.KeyValues() {
		require.Equal(t, expectedObjs[i], o)
	}

	// test iteration descending
	reverseIter := m.Iterate(ctx, keys.NewRange[keys.StringKey]().Descending())
	defer reverseIter.Close()
	for i, o := range reverseIter.KeyValues() {
		require.Equal(t, expectedObjs[len(expectedObjs)-1-i], o)
	}

	// test key iteration
	keyIter := m.Iterate(ctx, keys.NewRange[keys.StringKey]())
	defer keyIter.Close()
	for i, o := range keyIter.Keys() {
		require.Equal(t, expectedObjs[i].Key, o)
	}

	valIter := m.Iterate(ctx, keys.NewRange[keys.StringKey]())
	defer valIter.Close()
	for i, o := range valIter.Values() {
		require.Equal(t, expectedObjs[i].Value, o)
	}
}
