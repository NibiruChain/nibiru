package collections

import (
	"testing"

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
