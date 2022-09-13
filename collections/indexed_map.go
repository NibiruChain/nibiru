package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

type Indexes[K keys.Key, V any] interface {
	IndexList() []Index[K, V]
}

type Index[K keys.Key, V any] interface {
	Insert(ctx sdk.Context, k K, v V)
	Delete(ctx sdk.Context, k K, v V)
	SetPrefix([]byte)
}

type MultiIndex[IK keys.Key, PK keys.Key, V any] struct {
	prefix  []byte
	sk      sdk.StoreKey
	indexFn func(V) IK
}

func (i MultiIndex[IK, PK, V]) Insert(ctx sdk.Context, k PK, v V) {

}

func (i MultiIndex[IK, PK, V]) Delete(ctx sdk.Context, k PK, v V) {

}

func (i *MultiIndex[IK, PK, V]) SetPrefix(pfx []byte) {
	i.prefix = pfx
}

func NewMultiIndex[IK keys.Key, PK keys.Key, V any](sk sdk.StoreKey, indexFn func(V) IK) *MultiIndex[IK, PK, V] {
	return &MultiIndex[IK, PK, V]{
		prefix:  nil,
		sk:      nil,
		indexFn: nil,
	}
}

func NewIndexedMap[K keys.Key, V any, PV interface {
	*V
	Object
}, I Indexes[K, V]](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8, indexes I) IndexedMap[K, V, PV, I] {
	m := NewMap[K, V, PV](cdc, sk, 0)
	m.prefix = []byte{prefix, 0} // we set the prefix to provided prefix + 0
	for i, index := range indexes.IndexList() {
		index.SetPrefix([]byte{prefix, uint8(i) + 1})
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
