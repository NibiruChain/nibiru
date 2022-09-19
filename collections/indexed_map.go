package collections

import (
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Indexes defines a structure which contains indexes definitions.
type Indexes[K keys.Key, V any] interface {
	// IndexList must be implemented by the indexes struct and must return
	// the indexes to query.
	// NOTE: changing the order at which elements in IndexList are provided
	// is breaking, and will cause state corruption.
	IndexList() []Index[K, V]
}

// Index defines the index API needed by IndexedMap
// to index objects of type V, with primary key PK.
type Index[PK keys.Key, V any] interface {
	// Insert inserts elements in the index.
	Insert(ctx sdk.Context, k PK, v V)
	// Delete deletes objects from the index.
	Delete(ctx sdk.Context, k PK, v V)
	// Initialize is called by IndexedMap to initialize the Index.
	// id is provided by the indexed map based
	Initialize(cdc codec.BinaryCodec, storeKey sdk.StoreKey, id uint8)
}

func NewIndexedMap[K keys.Key, V any, PV interface {
	*V
	Object
}, I Indexes[K, V]](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8, indexes I) IndexedMap[K, V, PV, I] {
	m := NewMap[K, V, PV](cdc, sk, 0)
	m.prefix = []byte{prefix, 0}
	for i, index := range indexes.IndexList() {
		index.Initialize(cdc, sk, uint8(i)+1)
	}

	return IndexedMap[K, V, PV, I]{
		Indexes: indexes,
		m:       m,
	}
}

type IndexedMap[K keys.Key, V any, PV interface {
	*V
	Object
}, I Indexes[K, V]] struct {
	Indexes I
	m       Map[K, V, PV]
}

func (i IndexedMap[K, V, PV, I]) Insert(ctx sdk.Context, k K, v V) {
	i.m.Insert(ctx, k, v)
	for _, index := range i.Indexes.IndexList() {
		index.Insert(ctx, k, v)
	}
}

func (i IndexedMap[K, V, PV, I]) Delete(ctx sdk.Context, k K) error {
	old, err := i.m.Get(ctx, k)
	if err != nil {
		return err
	}
	err = i.m.Delete(ctx, k)
	if err != nil {
		panic(err)
	}

	for _, index := range i.Indexes.IndexList() {
		index.Delete(ctx, k, old)
	}
	return nil
}

func (i IndexedMap[K, V, PV, I]) Get(ctx sdk.Context, k K) (V, error) {
	return i.m.Get(ctx, k)
}

func (i IndexedMap[K, V, PV, I]) GetOr(ctx sdk.Context, k K, def V) V {
	return i.m.GetOr(ctx, k, def)
}

func (i IndexedMap[K, V, PV, I]) Iterate(ctx sdk.Context, rng keys.Range[K]) MapIterator[K, V, PV] {
	return i.m.Iterate(ctx, rng)
}
