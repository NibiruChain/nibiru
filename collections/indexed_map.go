package collections

import (
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IndexersProvider is implemented by structs containing
// a series of Indexer instances.
type IndexersProvider[PK keys.Key, V any] interface {
	// IndexerList provides the list of Indexer contained
	// in the struct.
	IndexerList() []Indexer[PK, V]
}

// Indexer defines an object which given an object V
// and a primary key PK, creates a relationship
// between one or multiple fields of the object V
// with the primary key PK.
type Indexer[PK keys.Key, V any] interface {
	// Insert is called when the IndexedMap is inserting
	// an object into its state, so the Indexer here
	// creates the relationship between primary key
	// and the fields of the object V.
	Insert(ctx sdk.Context, primaryKey PK, v V)
	// Delete is called when the IndexedMap is removing
	// the object V and hence the relationship between
	// V and its primary keys need to be removed too.
	Delete(ctx sdk.Context, primaryKey PK, v V)
}

// NewIndexedMap instantiates a new IndexedMap instance.
func NewIndexedMap[PK keys.Key, V any, PV interface {
	*V
	Object
}, I IndexersProvider[PK, V]](cdc codec.BinaryCodec, storeKey sdk.StoreKey, namespace uint8, indexers I) IndexedMap[PK, V, PV, I] {
	m := NewMap[PK, V, PV](cdc, storeKey, namespace)
	m.prefix = append(m.prefix, 0)
	return IndexedMap[PK, V, PV, I]{
		m:       m,
		Indexes: indexers,
	}
}

// IndexedMap defines a map which is indexed using the IndexersProvider
// PK defines the primary key of the object V.
type IndexedMap[PK keys.Key, V any, PV interface {
	*V
	Object
}, I IndexersProvider[PK, V]] struct {
	m       Map[PK, V, PV] // maintains PrimaryKey (PK) -> Object (V) bytes
	Indexes I              // maintains relationship between indexing keys and PrimaryKey (PK)
}

// Get returns the object V given its primary key PK.
func (i IndexedMap[PK, V, PV, I]) Get(ctx sdk.Context, key PK) (V, error) {
	return i.m.Get(ctx, key)
}

// GetOr returns the object V given its primary key PK, or if the operation fails
// returns the provided default.
func (i IndexedMap[PK, V, PV, I]) GetOr(ctx sdk.Context, key PK, def V) V {
	return i.m.GetOr(ctx, key, def)
}

// Insert inserts the object v into the Map using the primary key, then
// iterates over every registered Indexer and instructs them to create
// the relationship between the primary key PK and the object v.
func (i IndexedMap[PK, V, PV, I]) Insert(ctx sdk.Context, key PK, v V) {
	// before inserting we need to assert if another instance of this
	// primary key exist in order to remove old relationships from indexes.
	old, err := i.m.Get(ctx, key)
	if err == nil {
		i.unindex(ctx, key, old)
	}
	// insert and index
	i.m.Insert(ctx, key, v)
	i.index(ctx, key, v)
}

// Delete fetches the object from the Map removes it from the Map
// then instructs every Indexer to remove the relationships between
// the object and the associated primary keys.
func (i IndexedMap[PK, V, PV, I]) Delete(ctx sdk.Context, key PK) error {
	// we prefetch the object
	v, err := i.m.Get(ctx, key)
	if err != nil {
		return err
	}
	err = i.m.Delete(ctx, key)
	if err != nil {
		// this must never happen
		panic(err)
	}
	i.unindex(ctx, key, v)
	return nil
}

// Iterate iterates over the underlying store containing the concrete objects.
// The range provided filters over the primary keys.
func (i IndexedMap[PK, V, PV, I]) Iterate(ctx sdk.Context, rng keys.Range[PK]) MapIterator[PK, V, PV] {
	return i.m.Iterate(ctx, rng)
}

func (i IndexedMap[PK, V, PV, I]) index(ctx sdk.Context, key PK, v V) {
	for _, indexer := range i.Indexes.IndexerList() {
		indexer.Insert(ctx, key, v)
	}
}

func (i IndexedMap[PK, V, PV, I]) unindex(ctx sdk.Context, key PK, v V) {
	for _, indexer := range i.Indexes.IndexerList() {
		indexer.Delete(ctx, key, v)
	}
}

// ------------------------------------------------- indexers ----------------------------------------------------------

// NewMultiIndex instantiates a new MultiIndex instance. Namespace must match the namespace provided
// to the NewIndexedMap function. IndexID must be unique across every Indexer contained in the IndexersProvider
// provided to the NewIndexedMap function. IndexID must be different from 0.
// getIndexingKeyFunc is a function which given the object returns the key we use to index the object.
func NewMultiIndex[IK, PK keys.Key, V any](cdc codec.BinaryCodec, sk sdk.StoreKey, namespace uint8, indexID uint8, getIndexingKeyFunc func(v V) IK) MultiIndex[IK, PK, V] {
	if indexID == 0 {
		panic("invalid index id cannot be equal to 0")
	}
	ks := NewKeySet[keys.Pair[IK, PK]](cdc, sk, namespace)
	ks.prefix = append(ks.prefix, indexID)

	return MultiIndex[IK, PK, V]{
		secondaryKeys:  ks,
		getIndexingKey: getIndexingKeyFunc,
	}
}

// MultiIndex defines an Indexer with no uniqueness constraints.
// Meaning that given two objects V1 and V2 both can be indexed
// with the same secondary key.
// Example:
// Person1 { ID: 0, City: Milan }
// Person2 { ID: 1, City: Milan }
// Both can be indexed with the secondary key "Milan".
// The key generated are, respectively:
// keys.Pair[Milan, 0]
// keys.Pair[Milan, 1]
// So if we want to get all the objects whose City is Milan
// we prefix over keys.Pair[Milan, nil], and we get the respective primary keys: 0,1.
type MultiIndex[IK, PK keys.Key, V any] struct {
	secondaryKeys KeySet[keys.Pair[IK, PK]]
	// getIndexingKey is a function which provided the object, returns the indexing key
	getIndexingKey func(v V) IK
}

// Insert implements the Indexer interface.
func (i MultiIndex[IK, PK, V]) Insert(ctx sdk.Context, pk PK, v V) {
	indexingKey := i.getIndexingKey(v)
	i.secondaryKeys.Insert(ctx, keys.Join(indexingKey, pk))
}

// Delete implements the Indexer interface.
func (i MultiIndex[IK, PK, V]) Delete(ctx sdk.Context, pk PK, v V) {
	indexingKey := i.getIndexingKey(v)
	i.secondaryKeys.Delete(ctx, keys.Join(indexingKey, pk))
}

// Iterate iterates over the provided range.
func (i MultiIndex[IK, PK, V]) Iterate(ctx sdk.Context, rng keys.Range[keys.Pair[IK, PK]]) IndexerIterator[IK, PK] {
	iter := i.secondaryKeys.Iterate(ctx, rng)
	return (IndexerIterator[IK, PK])(iter)
}

// ExactMatch returns an iterator of all the primary keys of objects which contain
// the provided indexing key ik.
func (i MultiIndex[IK, PK, V]) ExactMatch(ctx sdk.Context, ik IK) IndexerIterator[IK, PK] {
	return i.Iterate(ctx, keys.PairRange[IK, PK]{}.Prefix(ik))
}

// ReverseExactMatch works in the same way as ExactMatch, but the iteration happens in reverse.
func (i MultiIndex[IK, PK, V]) ReverseExactMatch(ctx sdk.Context, ik IK) IndexerIterator[IK, PK] {
	return i.Iterate(ctx, keys.PairRange[IK, PK]{}.Prefix(ik).Descending())
}

// IndexerIterator wraps a KeySetIterator to provide more useful functionalities
// around index key iteration.
type IndexerIterator[IK, PK keys.Key] KeySetIterator[keys.Pair[IK, PK]]

// FullKey returns the iterator current key composed of both indexing key and primary key.
func (i IndexerIterator[IK, PK]) FullKey() keys.Pair[IK, PK] {
	return (KeySetIterator[keys.Pair[IK, PK]])(i).Key()
}

// FullKeys fully consumes the iterator and returns the set of joined indexing key and primary key found.
func (i IndexerIterator[IK, PK]) FullKeys() []keys.Pair[IK, PK] {
	return (KeySetIterator[keys.Pair[IK, PK]])(i).Keys()
}

// PrimaryKey returns the iterator current primary key
func (i IndexerIterator[IK, PK]) PrimaryKey() PK { return i.FullKey().K2() }

// PrimaryKeys fully consumes the iterator and returns the set of primary keys found.
func (i IndexerIterator[IK, PK]) PrimaryKeys() []PK {
	ks := i.FullKeys()
	pks := make([]PK, len(ks))
	for i, k := range ks {
		pks[i] = k.K2()
	}
	return pks
}

func (i IndexerIterator[IK, PK]) Next()       { (KeySetIterator[keys.Pair[IK, PK]])(i).Next() }
func (i IndexerIterator[IK, PK]) Valid() bool { return (KeySetIterator[keys.Pair[IK, PK]])(i).Valid() }
func (i IndexerIterator[IK, PK]) Close()      { (KeySetIterator[keys.Pair[IK, PK]])(i).Close() }
