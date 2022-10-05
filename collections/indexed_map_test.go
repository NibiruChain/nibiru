package collections

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/collections/keys"
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

type indexes struct {
	City MultiIndex[keys.StringKey, keys.Uint64Key, person]
}

func (i indexes) IndexerList() []Indexer[keys.Uint64Key, person] {
	return []Indexer[keys.Uint64Key, person]{i.City}
}

func TestIndexedMap(t *testing.T) {
	sk, ctx, cdc := deps()
	m := NewIndexedMap[keys.Uint64Key, person, *person, indexes](cdc, sk, 0, indexes{
		City: NewMultiIndex[keys.StringKey, keys.Uint64Key, person](cdc, sk, 1, func(v person) keys.StringKey {
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
