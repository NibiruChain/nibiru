package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections/keys"
)

func TestKeyset(t *testing.T) {
	sk, ctx, cdc := deps()
	keyset := NewKeySet[keys.StringKey](cdc, sk, 0)

	key := keys.String("id")

	// test insert and get
	keyset.Insert(ctx, key)
	require.True(t, keyset.Has(ctx, key))

	// test delete and get error
	keyset.Delete(ctx, key)
	require.False(t, keyset.Has(ctx, key))
}

func TestKeyset_Iterate(t *testing.T) {
	sk, ctx, cdc := deps()
	keyset := NewKeySet[keys.StringKey](cdc, sk, 0)
	keyset.Insert(ctx, "a")
	keyset.Insert(ctx, "aa")
	keyset.Insert(ctx, "b")
	keyset.Insert(ctx, "bb")

	expectedKeys := []keys.StringKey{"a", "aa", "b", "bb"}

	iter := keyset.Iterate(ctx, keys.NewRange[keys.StringKey]())
	defer iter.Close()
	for i, o := range iter.Keys() {
		require.Equal(t, expectedKeys[i], o)
	}
}

func TestKeysetIterator(t *testing.T) {
	sk, ctx, cdc := deps()

	keyset := NewKeySet[keys.StringKey](cdc, sk, 0)
	keyset.Insert(ctx, "a")

	iter := keyset.Iterate(ctx, keys.NewRange[keys.StringKey]())
	defer iter.Close()

	assert.True(t, iter.Valid())
	assert.EqualValues(t, "a", iter.Key())

	iter.Next()
	assert.False(t, iter.Valid())
}
