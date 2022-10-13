package collections_test

import (
	"github.com/NibiruChain/nibiru/collections"
	"testing"

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

func TestMap(t *testing.T) {
	sk, ctx, _ := deps()
	m := collections.NewMap[string, string](sk, 0, collections.Keys.String, stringValue{})

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
	require.ErrorIs(t, err, collections.ErrNotFound)

	// test delete errors not exist
	err = m.Delete(ctx, key)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestMapGetOrDefault(t *testing.T) {
	sk, ctx, _ := deps()
	m := collections.NewMap[string, string](sk, 0, collections.Keys.String, stringValue{})
	assert.EqualValues(t, "default", m.GetOr(ctx, "foo", "default"))

	m.Insert(ctx, "foo", "not-default")
	assert.EqualValues(t, "not-default", m.GetOr(ctx, "foo", "default"))
}

func TestMapIterate(t *testing.T) {
	kv := func(o string) collections.KeyValue[string, string] {
		return collections.KeyValue[string, string]{
			Key:   o,
			Value: o,
		}
	}
	sk, ctx, _ := deps()
	m := collections.NewMap[string, string](sk, 0, collections.Keys.String, stringValue{})

	expectedObjs := []collections.KeyValue[string, string]{
		kv("a"), kv("aa"), kv("b"), kv("bb"),
	}

	m.Insert(ctx, "a", "a")
	m.Insert(ctx, "aa", "aa")
	m.Insert(ctx, "b", "b")
	m.Insert(ctx, "bb", "bb")

	// test iteration ascending
	iter := m.Iterate(ctx, collections.Range[string]{})
	defer iter.Close()
	for i, o := range iter.KeyValues() {
		require.Equal(t, expectedObjs[i], o)
	}

	// test iteration descending
	reverseIter := m.Iterate(ctx, collections.Range[string]{}.Descending())
	defer reverseIter.Close()
	for i, o := range reverseIter.KeyValues() {
		require.Equal(t, expectedObjs[len(expectedObjs)-1-i], o)
	}

	// test key iteration
	keyIter := m.Iterate(ctx, collections.Range[string]{})
	defer keyIter.Close()
	for i, o := range keyIter.Keys() {
		require.Equal(t, expectedObjs[i].Key, o)
	}

	valIter := m.Iterate(ctx, collections.Range[string]{})
	defer valIter.Close()
	for i, o := range valIter.Values() {
		require.Equal(t, expectedObjs[i].Value, o)
	}
}
