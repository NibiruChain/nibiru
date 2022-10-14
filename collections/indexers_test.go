package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiIndex(t *testing.T) {
	sk, ctx, _ := deps()
	im := NewMultiIndex[string, uint64, person](
		sk, 0,
		StringKeyEncoder, Uint64KeyEncoder,
		func(v person) string { return v.City },
	)
	// test insertions

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
	require.Equal(t, []uint64{0, 1}, ks)
	// test ReverseExactMatch
	ks = im.ReverseExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []uint64{1, 0}, ks)
	// test after removal it is not present
	im.Delete(ctx, persons[0].ID, persons[0])
	ks = im.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []uint64{1}, ks)

	// test iteration
	iter := im.Iterate(ctx, PairRange[string, uint64]{}.Descending())
	fk := iter.FullKey()
	require.Equal(t, fk.K1(), "new york")
	require.Equal(t, fk.K2(), uint64(2))
}

func TestIndexerIterator(t *testing.T) {
	sk, ctx, _ := deps()
	// test insertions
	im := NewMultiIndex[string, uint64, person](
		sk, 0,
		StringKeyEncoder, Uint64KeyEncoder,
		func(v person) string { return v.City },
	)

	im.Insert(ctx, 0, person{ID: 0, City: "milan"})
	im.Insert(ctx, 1, person{ID: 1, City: "milan"})

	iter := im.Iterate(ctx, PairRange[string, uint64]{})
	defer iter.Close()

	require.Equal(t, Join[string, uint64]("milan", 0), iter.FullKey())
	require.Equal(t, uint64(0), iter.PrimaryKey())

	// test next
	iter.Next()
	require.Equal(t, uint64(1), iter.PrimaryKey())

	require.Equal(t, iter.Valid(), true)
	iter.Next()
	require.False(t, iter.Valid())
}
