package collections

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IndexersProvider is implemented by structs containing
// a series of Indexer instances.
type IndexersProvider[PK any, V any] interface {
	// IndexerList provides the list of Indexer contained
	// in the struct.
	IndexerList() []Indexer[PK, V]
}

// Indexer defines an object which given an object V
// and a primary key PK, creates a relationship
// between one or multiple fields of the object V
// with the primary key PK.
type Indexer[PK any, V any] interface {
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
func NewIndexedMap[PK any, V any, I IndexersProvider[PK, V]](
	storeKey sdk.StoreKey, namespace Namespace,
	primaryKeyEncoder KeyEncoder[PK],
	valueEncoder ValueEncoder[V],
	indexers I) IndexedMap[PK, V, I] {
	m := NewMap[PK, V](storeKey, namespace, primaryKeyEncoder, valueEncoder)
	return IndexedMap[PK, V, I]{
		m:       m,
		Indexes: indexers,
	}
}

// IndexedMap defines a map which is indexed using the IndexersProvider
// PK defines the primary key of the object V.
type IndexedMap[PK any, V any, I IndexersProvider[PK, V]] struct {
	m       Map[PK, V] // maintains PrimaryKey (PK) -> Object (V) bytes
	Indexes I          // struct that groups together Indexer instances, implements IndexersProvider
}

// Get returns the object V given its primary key PK.
func (i IndexedMap[PK, V, I]) Get(ctx sdk.Context, key PK) (V, error) {
	return i.m.Get(ctx, key)
}

// GetOr returns the object V given its primary key PK, or if the operation fails
// returns the provided default.
func (i IndexedMap[PK, V, I]) GetOr(ctx sdk.Context, key PK, def V) V {
	return i.m.GetOr(ctx, key, def)
}

// Insert inserts the object v into the Map using the primary key, then
// iterates over every registered Indexer and instructs them to create
// the relationship between the primary key PK and the object v.
func (i IndexedMap[PK, V, I]) Insert(ctx sdk.Context, key PK, v V) {
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
func (i IndexedMap[PK, V, I]) Delete(ctx sdk.Context, key PK) error {
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
func (i IndexedMap[PK, V, I]) Iterate(ctx sdk.Context, rng Range[PK]) Iterator[PK, V] {
	return i.m.Iterate(ctx, rng)
}

func (i IndexedMap[PK, V, I]) index(ctx sdk.Context, key PK, v V) {
	for _, indexer := range i.Indexes.IndexerList() {
		indexer.Insert(ctx, key, v)
	}
}

func (i IndexedMap[PK, V, I]) unindex(ctx sdk.Context, key PK, v V) {
	for _, indexer := range i.Indexes.IndexerList() {
		indexer.Delete(ctx, key, v)
	}
}
