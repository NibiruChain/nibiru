package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type person struct {
	ID   uint64
	City string
}

type indexes struct {
	City MultiIndex[string, uint64, person]
}

func (i indexes) IndexerList() []Indexer[uint64, person] {
	return []Indexer[uint64, person]{i.City}
}

func TestIndexedMap(t *testing.T) {
	sk, ctx, _ := deps()
	m := NewIndexedMap[uint64, person, indexes](
		sk, 0,
		Uint64KeyEncoder, jsonValue[person]{},
		indexes{
			City: NewMultiIndex[string, uint64, person](sk, 1,
				StringKeyEncoder, Uint64KeyEncoder,
				func(v person) string {
					return v.City
				}),
		},
	)

	m.Insert(ctx, 0, person{ID: 0, City: "milan"})
	m.Insert(ctx, 1, person{ID: 1, City: "new york"})
	m.Insert(ctx, 2, person{ID: 2, City: "milan"})

	// correct insertion
	res := m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []uint64{0, 2}, res)

	// once deleted, it's removed from indexes
	err := m.Delete(ctx, 0)
	require.NoError(t, err)
	res = m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []uint64{2}, res)

	// insertion on an already existing primary key
	// clears the old indexes, hence PK 2 => city "milan"
	// is now converted to PK 2 => city "new york"
	m.Insert(ctx, 2, person{ID: 2, City: "new york"})
	require.Empty(t, m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys())
	res = m.Indexes.City.ExactMatch(ctx, "new york").PrimaryKeys()
	require.Equal(t, []uint64{1, 2}, res)

	// test ordinary map functionality
	p, err := m.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, person{2, "new york"}, p)

	p = m.GetOr(ctx, 10, person{10, "sf"})
	require.Equal(t, p, person{10, "sf"})

	persons := m.Iterate(ctx, Range[uint64]{}).Values()
	require.Equal(t, []person{{1, "new york"}, {2, "new york"}}, persons)
}
