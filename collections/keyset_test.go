package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeySet(t *testing.T) {
	sk, ctx, _ := deps()
	keyset := NewKeySet[string](sk, 0, StringKeyEncoder)

	// test insert and get
	key := "hi"
	keyset.Insert(ctx, key)
	require.True(t, keyset.Has(ctx, key))

	// test delete and get error
	keyset.Delete(ctx, key)
	require.False(t, keyset.Has(ctx, key))
}

func TestKeySet_Iterate(t *testing.T) {
	sk, ctx, _ := deps()
	keyset := NewKeySet[string](sk, 0, StringKeyEncoder)
	keyset.Insert(ctx, "a")
	keyset.Insert(ctx, "aa")
	keyset.Insert(ctx, "b")
	keyset.Insert(ctx, "bb")

	expectedKeys := []string{"a", "aa", "b", "bb"}

	iter := keyset.Iterate(ctx, Range[string]{})
	defer iter.Close()
	for i, o := range iter.Keys() {
		require.Equal(t, expectedKeys[i], o)
	}
}

func TestKeysetIterator(t *testing.T) {
	sk, ctx, _ := deps()

	keyset := NewKeySet[string](sk, 0, StringKeyEncoder)
	keyset.Insert(ctx, "a")

	iter := keyset.Iterate(ctx, Range[string]{})
	defer iter.Close()

	assert.True(t, iter.Valid())
	assert.EqualValues(t, "a", iter.Key())

	iter.Next()
	assert.False(t, iter.Valid())
}
