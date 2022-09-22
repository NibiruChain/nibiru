package collections

import (
	"encoding/json"
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

type person struct {
	ID   keys.Uint64Key
	City keys.StringKey
}

func (p person) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *person) Unmarshal(b []byte) error {
	return json.Unmarshal(b, &p)
}

func TestMultiIndex(t *testing.T) {
	sk, ctx, cdc := deps()
	// test panics if indexID is zero
	require.Panics(t, func() {
		_ = NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, 0, func(v person) keys.StringKey {
			return v.City
		})
	})

	// test insertions
	im := NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, 1, func(v person) keys.StringKey {
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
	im := NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, 1, func(v person) keys.StringKey {
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

type indexes struct {
	City MultiIndex[keys.StringKey, keys.Uint64Key, person]
}

func (i indexes) IndexerList() []Indexer[keys.Uint64Key, person] {
	return []Indexer[keys.Uint64Key, person]{i.City}
}

func TestIndexedMap(t *testing.T) {
	sk, ctx, cdc := deps()
	m := NewIndexedMap[keys.Uint64Key, person, *person, indexes](cdc, sk, 0, indexes{
		City: NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 0, 1, func(v person) keys.StringKey {
			return v.City
		}),
	})

	m.Insert(ctx, 0, person{ID: 0, City: "milan"})
	m.Insert(ctx, 1, person{ID: 1, City: "new york"})
	m.Insert(ctx, 2, person{ID: 2, City: "milan"})

	// correct insertion
	res := m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{0, 2}, res)

	// once deleted, it's removed from indexes
	err := m.Delete(ctx, 0)
	require.NoError(t, err)
	res = m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{2}, res)

	// insertion on an already existing primary key
	// clears the old indexes, hence PK 2 => city "milan"
	// is now converted to PK 2 => city "new york"
	m.Insert(ctx, 2, person{ID: 2, City: "new york"})
	require.Empty(t, m.Indexes.City.ExactMatch(ctx, "milan").PrimaryKeys())
	res = m.Indexes.City.ExactMatch(ctx, "new york").PrimaryKeys()
	require.Equal(t, []keys.Uint64Key{1, 2}, res)

	// test ordinary map functionality
	p, err := m.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, person{2, "new york"}, p)

	p = m.GetOr(ctx, 10, person{10, "sf"})
	require.Equal(t, p, person{10, "sf"})

	persons := m.Iterate(ctx, keys.NewRange[keys.Uint64Key]()).Values()
	require.Equal(t, []person{{1, "new york"}, {2, "new york"}}, persons)
}
