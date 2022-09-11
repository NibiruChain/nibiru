package collections

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/collections/keys/bound"
)

type Order uint8

const (
	OrderAscending Order = iota
	OrderDescending
)

func NewMap[K keys.Key, V any, PV interface {
	*V
	Object
}](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Map[K, V, PV] {
	return Map[K, V, PV]{
		cdc:    cdc,
		sk:     sk,
		prefix: []byte{prefix},
	}
}

// Map defines a collection which simply does mappings between primary keys and objects.
type Map[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	cdc    codec.BinaryCodec
	sk     sdk.StoreKey
	prefix []byte
	_      K
	_      V
}

func (m Map[K, V, PV]) getStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(m.sk), m.prefix)
}

func (m Map[K, V, PV]) Insert(ctx sdk.Context, key K, object V) {
	store := m.getStore(ctx)
	store.Set(key.PrimaryKey(), m.cdc.MustMarshal(PV(&object)))
}

func (m Map[K, V, PV]) Get(ctx sdk.Context, key K) (V, error) {
	store := m.getStore(ctx)
	pk := key.PrimaryKey()
	bytes := store.Get(pk)
	if bytes == nil {
		var x V
		return x, ErrNotFound
	}

	x := new(V)
	m.cdc.MustUnmarshal(bytes, PV(x))
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
	pk := key.PrimaryKey()
	if !store.Has(pk) {
		return fmt.Errorf("not found")
	}

	store.Delete(pk)
	return nil
}

func (m Map[K, V, PV]) Iterate(ctx sdk.Context, start bound.Bound, end bound.Bound, order Order) MapIterator[K, V, PV] {
	store := m.getStore(ctx)
	return newMapIterator[K, V, PV](m.cdc, store, start, end, order)
}

// Prefix returns a Prefix instance over a key.
func (m Map[K, V, PV]) Prefix(ctx sdk.Context, p K) Prefix[K, V, PV] {
	bytes := p.PrimaryKey()
	return Prefix[K, V, PV]{
		cdc:    m.cdc,
		prefix: prefix.NewStore(m.getStore(ctx), bytes),
	}
}

func (m Map[K, V, PV]) GetAll(ctx sdk.Context) []V {
	iter := m.Iterate(ctx, bound.None, bound.None, OrderAscending)
	defer iter.Close()

	var list []V
	for ; iter.Valid(); iter.Next() {
		list = append(list, iter.Value())
	}

	return list
}

func newMapIterator[K keys.Key, V any, PV interface {
	*V
	Object
}](cdc codec.BinaryCodec, store sdk.KVStore, start, end bound.Bound, order Order) MapIterator[K, V, PV] {
	startBytes := start.Bytes()
	endBytes := end.Bytes()
	switch order {
	case OrderAscending:
		return MapIterator[K, V, PV]{
			cdc:  cdc,
			iter: store.Iterator(startBytes, endBytes),
		}
	case OrderDescending:
		return MapIterator[K, V, PV]{
			cdc:  cdc,
			iter: store.ReverseIterator(startBytes, endBytes),
		}
	default:
		panic(fmt.Errorf("unrecognized order"))
	}
}

type MapIterator[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	cdc  codec.BinaryCodec
	iter sdk.Iterator
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
	i.cdc.MustUnmarshal(i.iter.Value(), x)
	return *x
}

func (i MapIterator[K, V, PV]) Key() K {
	var k K
	return k.FromPrimaryKeyBytes(i.iter.Key()).(K) // TODO implement this better
}

type Prefix[K keys.Key, V any, PV interface {
	*V
	Object
}] struct {
	_      K
	_      V
	_      PV
	cdc    codec.BinaryCodec
	prefix sdk.KVStore
}

func (p Prefix[K, V, PV]) Iterate(start, end bound.Bound, order Order) MapIterator[K, V, PV] {
	startBytes := start.Bytes()
	endBytes := end.Bytes()
	switch order {
	case OrderAscending:
		return MapIterator[K, V, PV]{
			cdc:  p.cdc,
			iter: p.prefix.Iterator(startBytes, endBytes),
		}
	case OrderDescending:
		return MapIterator[K, V, PV]{
			cdc:  p.cdc,
			iter: p.prefix.ReverseIterator(startBytes, endBytes),
		}
	default:
		panic(fmt.Errorf("unrecognized order"))
	}
}
