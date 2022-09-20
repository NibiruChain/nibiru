package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

// MultiIndex represents an index in which there is no uniqueness constraint.
// Which means that multiple primary keys with the same key can exist.
// It is implemented using a KeySet with keys.Pair[IK, PK], where
// IK is the index key and PK is the primary key of the object.
// Indexing keys are simple references, meaning that
// the Indexing key is formed as concat(index_key, primary_key)
// Example, given an object Obj{City: milan, ID: 0}, where City is the index and ID is the primary key
// The following is the generated KeyPair
// keys.Pair[K1: milan, K2: 0]
// Simulating that there are multiple objects that were indexed, the following is the Raw KV mapping
// Key                   | Value
// ('milan', 0)          | []byte{}
// ('milan', 5)          | []byte{}
// ('new york', 1)       | []byte{}
// ('new york', 2)       | []byte{}
// So if we want to get all the objects which had City as 'milan'
// we would prefix over 'milan' in the raw KV to get all the primary keys => 0, 5.
type MultiIndex[IK keys.Key, PK keys.Key, V any] struct {
	// indexFn is used to get the secondary key (aka index key)
	// from the object we're indexing.
	indexFn func(V) IK
	// secondaryKeys is a multipart key composed by the
	// index key (IK) and the primary key (PK)
	secondaryKeys KeySet[keys.Pair[IK, PK]]
}

// Insert inserts fetches the index key IK from the object v.
// And then maps the index key to the primary key.
func (i *MultiIndex[IK, PK, V]) Insert(ctx sdk.Context, pk PK, v V) {
	// get secondary key
	sk := i.indexFn(v)
	// insert it
	i.secondaryKeys.Insert(ctx, keys.Join(sk, pk))
}

// Delete removes the object from the KeySet, removing the references
// of PK from the index.
func (i *MultiIndex[IK, PK, V]) Delete(ctx sdk.Context, pk PK, v V) {
	sk := i.indexFn(v)
	i.secondaryKeys.Delete(ctx, keys.Join(sk, pk))
}

// Initialize initializes the index, objectNamespace defines the broader object (V) namespace.
// IndexNamespace identifies the index namespace in the object namespace.
func (i *MultiIndex[IK, PK, V]) Initialize(cdc codec.BinaryCodec, storeKey sdk.StoreKey, objectNamespace uint8, indexNamespace uint8) {
	i.secondaryKeys = NewKeySet[keys.Pair[IK, PK]](cdc, storeKey, indexNamespace)
	i.secondaryKeys.prefix = []byte{objectNamespace, indexNamespace}
}

// Iterate iterates over indexing keys using the provided range.
func (i *MultiIndex[IK, PK, V]) Iterate(ctx sdk.Context, rng keys.Range[keys.Pair[IK, PK]]) IndexIterator[IK, PK] {
	return IndexIterator[IK, PK]{
		ks: i.secondaryKeys.Iterate(ctx, rng),
	}
}

func NewMultiIndex[IK keys.Key, PK keys.Key, V any](indexFn func(V) IK) *MultiIndex[IK, PK, V] {
	return &MultiIndex[IK, PK, V]{
		indexFn: indexFn,
	}
}

type IndexIterator[IK keys.Key, PK keys.Key] struct {
	ks KeySetIterator[keys.Pair[IK, PK]]
}

func (i IndexIterator[IK, PK]) Keys() []PK {
	keys := i.ks.Keys()
	primaryKeys := make([]PK, len(keys))
	for i, key := range keys {
		primaryKeys[i] = key.K2()
	}
	return primaryKeys
}

func (i IndexIterator[IK, PK]) FullKeys() []keys.Pair[IK, PK] {
	return i.ks.Keys()
}

func (i IndexIterator[IK, PK]) Key() PK {
	return i.FullKey().K2()
}

func (i IndexIterator[IK, PK]) FullKey() keys.Pair[IK, PK] {
	return i.ks.Key()
}

func (i IndexIterator[IK, PK]) Next() {
	i.ks.Next()
}

func (i IndexIterator[IK, PK]) Close() {
	i.ks.Close()
}

func (i IndexIterator[IK, PK]) Valid() bool {
	return i.ks.Valid()
}
