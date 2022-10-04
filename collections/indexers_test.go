package collections

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections/keys"
)

func TestMultiIndex(t *testing.T) {
	sk, ctx, cdc := deps()

	// test insertions
	im := NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, func(v person) keys.StringKey {
		return v.City
	})
	persons := []person{
		{
			ID:   0,
			City: "milan",
		},
		{
			ID:   1,
			City: "milan",
		},
		{
			ID:   2,
			City: "new york",
		},
	}

	for _, p := range persons {
		im.Insert(ctx, p.ID, p)
	}

	// test iterations and matches alongside PrimaryKeys ( and indirectly FullKeys )
	// test ExactMatch
	ks := im.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{0, 1}, ks)
	// test ReverseExactMatch
	ks = im.ReverseExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{1, 0}, ks)
	// test after removal it is not present
	im.Delete(ctx, persons[0].ID, persons[0])
	ks = im.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{1}, ks)

	// test iteration
	iter := im.Iterate(ctx, keys.PairRange[keys.StringKey, keys.Uint64Key]{}.Descending())
	fk := iter.FullKey()
	require.Equal(t, fk.K1(), keys.String("new york"))
	require.Equal(t, fk.K2(), keys.Uint64(uint64(2)))
}

func TestIndexerIterator(t *testing.T) {
	sk, ctx, cdc := deps()
	// test insertions
	im := NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, func(v person) keys.StringKey {
		return v.City
	})

	im.Insert(ctx, 0, person{ID: 0, City: "milan"})
	im.Insert(ctx, 1, person{ID: 1, City: "milan"})

	iter := im.Iterate(ctx, keys.PairRange[keys.StringKey, keys.Uint64Key]{})
	defer iter.Close()

	require.Equal(t, keys.Join[keys.StringKey, keys.Uint64Key]("milan", 0), iter.FullKey())
	require.Equal(t, keys.Uint64(uint64(0)), iter.PrimaryKey())

	// test next
	iter.Next()
	require.Equal(t, keys.Uint64(uint64(1)), iter.PrimaryKey())

	require.Equal(t, iter.Valid(), true)
	iter.Next()
	require.False(t, iter.Valid())
}
