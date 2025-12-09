package collections

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

type MapImpl[K, V any] interface {
	Delete(ctx sdk.Context, k K) error
	Get(ctx sdk.Context, k K) (v V, err error)
	GetOr(ctx sdk.Context, key K, def V) (v V)
	GetStore(ctx sdk.Context) sdk.KVStore
	Insert(ctx sdk.Context, k K, v V)
	Iterate(ctx sdk.Context, rng Ranger[K]) Iterator[K, V]
}

func TestMap(t *testing.T) {
	sk, ctx, _ := deps()
	RunTestMap(t, ctx, NewMap[string, string](sk, 0, StringKeyEncoder, stringValue{}))
	sk, ctx, _ = deps()
	RunTestMap(t, ctx, NewMapTransient[string, string](sk, 1, StringKeyEncoder, stringValue{}))
}

func TestMapGetOrDefault(t *testing.T) {
	sk, ctx, _ := deps()
	RunTestMapGetOrDefault(t, ctx, NewMap[string, string](sk, 2, StringKeyEncoder, stringValue{}))
	RunTestMapGetOrDefault(t, ctx, NewMapTransient[string, string](sk, 3, StringKeyEncoder, stringValue{}))
}

func TestMapIterate(t *testing.T) {
	sk, ctx, _ := deps()
	RunTestMapIterate(t, ctx, NewMap[string, string](sk, 0, StringKeyEncoder, stringValue{}))
	RunTestMapIterate(t, ctx, NewMapTransient[string, string](sk, 1, StringKeyEncoder, stringValue{}))
}

func RunTestMap(t *testing.T, ctx sdk.Context, m MapImpl[string, string]) {
	key := "id"
	expected := "test"

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
	require.ErrorContains(t, err, "test string") // assert value name is correctly reported
	require.ErrorContains(t, err, "id")          // assert key is correctly reported

	// test delete errors not exist
	err = m.Delete(ctx, key)
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "test string")
	require.ErrorContains(t, err, "id") // assert key is correctly reported
}

func RunTestMapGetOrDefault(t *testing.T, ctx sdk.Context, m MapImpl[string, string]) {
	assert.EqualValues(t, "default", m.GetOr(ctx, "foo", "default"))

	m.Insert(ctx, "foo", "not-default")
	assert.EqualValues(t, "not-default", m.GetOr(ctx, "foo", "default"))
}

func RunTestMapIterate(t *testing.T, ctx sdk.Context, m MapImpl[string, string]) {
	kv := func(o string) KeyValue[string, string] {
		return KeyValue[string, string]{
			Key:   o,
			Value: o,
		}
	}

	expectedObjs := []KeyValue[string, string]{
		kv("a"), kv("aa"), kv("b"), kv("bb"),
	}

	m.Insert(ctx, "a", "a")
	m.Insert(ctx, "aa", "aa")
	m.Insert(ctx, "b", "b")
	m.Insert(ctx, "bb", "bb")

	// test iteration ascending
	iter := m.Iterate(ctx, Range[string]{})
	defer iter.Close()
	for i, o := range iter.KeyValues() {
		require.Equal(t, expectedObjs[i], o)
	}

	// test iteration descending
	reverseIter := m.Iterate(ctx, Range[string]{}.Descending())
	defer reverseIter.Close()
	for i, o := range reverseIter.KeyValues() {
		require.Equal(t, expectedObjs[len(expectedObjs)-1-i], o)
	}

	// test key iteration
	keyIter := m.Iterate(ctx, Range[string]{})
	defer keyIter.Close()
	for i, o := range keyIter.Keys() {
		require.Equal(t, expectedObjs[i].Key, o)
	}

	valIter := m.Iterate(ctx, Range[string]{})
	defer valIter.Close()
	for i, o := range valIter.Values() {
		require.Equal(t, expectedObjs[i].Value, o)
	}
}
