package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

// MultiIndex represents an index in which there is no uniqueness constraint.
// Which means that multiple primary keys with the same key can exist.
type MultiIndex[IK keys.Key, PK keys.Key, V any] struct {
	indexFn func(V) IK
	// secondaryKeys is a multipart key composed by the
	// index key (IK) and the primary key (PK)
	secondaryKeys KeySet[keys.Pair[IK, PK]]
}

func (i *MultiIndex[IK, PK, V]) Insert(ctx sdk.Context, pk PK, v V) {
	// get secondary key
	sk := i.indexFn(v)
	// insert it
	i.secondaryKeys.Insert(ctx, keys.Join(sk, pk))
}

func (i *MultiIndex[IK, PK, V]) Delete(ctx sdk.Context, pk PK, v V) {
	sk := i.indexFn(v)
	i.secondaryKeys.Delete(ctx, keys.Join(sk, pk))
}

func (i *MultiIndex[IK, PK, V]) Initialize(cdc codec.BinaryCodec, storeKey sdk.StoreKey, id uint8) {
	i.secondaryKeys = NewKeySet[keys.Pair[IK, PK]](cdc, storeKey, id)
}

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
