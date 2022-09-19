package collections

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections/keys"
)

type object struct {
	ID    keys.Uint64Key
	Owner keys.StringKey
}

func (b object) Marshal() ([]byte, error) {
	return json.Marshal(b)
}

func (b *object) Unmarshal(x []byte) error {
	return json.Unmarshal(x, b)
}

type Index1 struct {
	Owner *MultiIndex[keys.StringKey, keys.Uint64Key, object]
}

func (i Index1) IndexList() []Index[keys.Uint64Key, object] {
	return []Index[keys.Uint64Key, object]{i.Owner}
}

func TestNewIndexedMap(t *testing.T) {
	sk, ctx, cdc := deps()
	im := NewIndexedMap[keys.Uint64Key, object, *object, Index1](cdc, sk, 0, Index1{
		Owner: NewMultiIndex[keys.StringKey, keys.Uint64Key, object](func(v object) keys.StringKey {
			return keys.String(v.Owner)
		}),
	})

	im.Insert(ctx, 0, object{
		ID:    0,
		Owner: keys.String("mercilex"),
	})

	im.Insert(ctx, 1, object{
		ID:    1,
		Owner: keys.String("mercilex"),
	})

	im.Insert(ctx, 2, object{
		ID:    2,
		Owner: keys.String("heisenberg"),
	})

	im.Insert(ctx, 3, object{
		ID:    3,
		Owner: "mercilex",
	})

	// we want to range over "mercilex" owned objects.
	rng := keys.NewRange[keys.Pair[keys.StringKey, keys.Uint64Key]]()
	pfx := keys.PairPrefix[keys.StringKey, keys.Uint64Key](keys.String("mercilex"))
	rng = rng.Prefix(pfx)

	ks := im.Indexes.Owner.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Uint64Key{0, 1, 3}, ks)

	// we want to range over "mercilex" owner objects, starting from key0 exclusive and ending key2 inclusive
	rng = keys.NewRange[keys.Pair[keys.StringKey, keys.Uint64Key]]()
	pfx = keys.PairPrefix[keys.StringKey, keys.Uint64Key](keys.String("mercilex"))
	rng = rng.Prefix(pfx)
	rng = rng.Start(keys.Exclusive(keys.PairSuffix[keys.StringKey, keys.Uint64Key](keys.Uint64(uint64(0)))))
	rng = rng.End(keys.Inclusive(keys.PairSuffix[keys.StringKey, keys.Uint64Key](keys.Uint64(uint64(2)))))

	ks = im.Indexes.Owner.Iterate(ctx, rng).Keys()
	require.Equal(t, []keys.Uint64Key{1}, ks)

	// removal of the element from the indexed map reflects on the indexes too
	require.NoError(t, im.Delete(ctx, ks[0]))
	ks = im.Indexes.Owner.Iterate(ctx, rng).Keys()
	require.Empty(t, ks)
}
