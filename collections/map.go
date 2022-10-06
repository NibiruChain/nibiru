package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
)

func NewMap[K keys.Key, V any, PV interface {
	*V
	Object
}](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Map[K, V, PV] {
	return Map[K, V, PV]{
		cdc:      newStoreCodec(cdc),
		sk:       sk,
		prefix:   []byte{prefix},
		typeName: typeName(PV(new(V))),
	}
}

// Map defines a collection which simply does mappings between primary keys and objects.
type Map[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	cdc      storeCodec
	sk       sdk.StoreKey
	prefix   []byte
	_        K
	_        V
	typeName string
}

func (m Map[K, V, PV]) getStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(m.sk), m.prefix)
}

func (m Map[K, V, PV]) Insert(ctx sdk.Context, key K, object V) {
	store := m.getStore(ctx)
	store.Set(key.KeyBytes(), m.cdc.marshal(PV(&object)))
}

func (m Map[K, V, PV]) Get(ctx sdk.Context, key K) (V, error) {
	store := m.getStore(ctx)
	pk := key.KeyBytes()
	bytes := store.Get(pk)
	if bytes == nil {
		var x V
		return x, notFoundError(m.typeName, key.String())
	}

	x := new(V)
	m.cdc.unmarshal(bytes, PV(x))
	return *x, nil
}

func (m Map[K, V, PV]) GetOr(ctx sdk.Context, key K, def V) V {
	got, err := m.Get(ctx, key)
	if err != nil {
		return def
	}

	return got
}

func (m Map[K, V, PV]) Delete(ctx sdk.Context, key K) error {
	store := m.getStore(ctx)
	pk := key.KeyBytes()
	if !store.Has(pk) {
		return notFoundError(m.typeName, key.String())
	}

	store.Delete(pk)
	return nil
}

func (m Map[K, V, PV]) Iterate(ctx sdk.Context, r keys.Range[K]) MapIterator[K, V, PV] {
	store := m.getStore(ctx)
	return newMapIterator[K, V, PV](m.cdc, store, r)
}

func newMapIterator[K keys.Key, V any, PV interface {
	*V
	Object
}](cdc storeCodec, store sdk.KVStore, r keys.Range[K]) MapIterator[K, V, PV] {
	iter, prefix := keys.IteratorFromRange(store, r)
	return MapIterator[K, V, PV]{
		prefix: prefix,
		cdc:    cdc,
		iter:   iter,
	}
}

type MapIterator[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	prefix []byte
	cdc    storeCodec
	iter   sdk.Iterator
}

func (i MapIterator[K, V, PV]) Close() {
	_ = i.iter.Close()
}

func (i MapIterator[K, V, PV]) Next() {
	i.iter.Next()
}

func (i MapIterator[K, V, PV]) Valid() bool {
	return i.iter.Valid()
}

func (i MapIterator[K, V, PV]) Value() V {
	x := PV(new(V))
	i.cdc.unmarshal(i.iter.Value(), x)
	return *x
}

func (i MapIterator[K, V, PV]) Key() K {
	var k K
	rawKey := append(i.prefix, i.iter.Key()...)
	_, c := k.FromKeyBytes(rawKey) // todo(mercilex): can we assert safety here?
	return c.(K)
}

// TODO doc
func (i MapIterator[K, V, PV]) Values() []V {
	defer i.Close()

	var values []V
	for ; i.iter.Valid(); i.iter.Next() {
		values = append(values, i.Value())
	}
	return values
}

// TODO doc
func (i MapIterator[K, V, PV]) Keys() []K {
	defer i.Close()

	var keys []K
	for ; i.iter.Valid(); i.iter.Next() {
		keys = append(keys, i.Key())
	}
	return keys
}

// todo doc
func (i MapIterator[K, V, PV]) KeyValues() []KeyValue[K, V, PV] {
	defer i.Close()

	var kvs []KeyValue[K, V, PV]
	for ; i.iter.Valid(); i.iter.Next() {
		kvs = append(kvs, KeyValue[K, V, PV]{
			Key:   i.Key(),
			Value: i.Value(),
		})
	}

	return kvs
}

type KeyValue[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	Key   K
	Value V
}
