package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

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
		pkPointers:     ks,
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
	// pkPointers is a KeySet of the joint indexing key and the primary key.
	// the generated keys always point to primary keys.
	pkPointers KeySet[keys.Pair[IK, PK]]
	// getIndexingKey is a function which provided the object, returns the indexing key
	getIndexingKey func(v V) IK
}

// Insert implements the Indexer interface.
func (i MultiIndex[IK, PK, V]) Insert(ctx sdk.Context, pk PK, v V) {
	indexingKey := i.getIndexingKey(v)
	i.pkPointers.Insert(ctx, keys.Join(indexingKey, pk))
}

// Delete implements the Indexer interface.
func (i MultiIndex[IK, PK, V]) Delete(ctx sdk.Context, pk PK, v V) {
	indexingKey := i.getIndexingKey(v)
	i.pkPointers.Delete(ctx, keys.Join(indexingKey, pk))
}

// Iterate iterates over the provided range.
func (i MultiIndex[IK, PK, V]) Iterate(ctx sdk.Context, rng keys.Range[keys.Pair[IK, PK]]) IndexerIterator[IK, PK] {
	iter := i.pkPointers.Iterate(ctx, rng)
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
