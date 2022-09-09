package collections

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Order uint8

const (
	OrderAscending Order = iota
	OrderDescending
)

type Object interface {
	codec.ProtoMarshaler
}

func NewMap[K Key, V any, PV interface {
	*V
	Object
}](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Map[K, V, PV] {
	return Map[K, V, PV]{
		cdc:    cdc,
		sk:     sk,
		prefix: []byte{prefix},
	}
}

type Map[K Key, V any, PV interface {
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

func (m Map[K, V, PV]) Delete(ctx sdk.Context, key K) error {
	store := m.getStore(ctx)
	pk := key.PrimaryKey()
	if !store.Has(pk) {
		return fmt.Errorf("not found")
	}

	store.Delete(pk)
	return nil
}

func (m Map[K, V, PV]) Iterate(ctx sdk.Context, start Bound, end Bound, order Order) Iterator[K, V, PV] {
	store := m.getStore(ctx)
	startBytes := start.bytes()
	endBytes := end.bytes()
	switch order {
	case OrderAscending:
		return Iterator[K, V, PV]{
			cdc:  m.cdc,
			iter: store.Iterator(startBytes, endBytes),
		}
	case OrderDescending:
		return Iterator[K, V, PV]{
			cdc:  m.cdc,
			iter: store.ReverseIterator(startBytes, endBytes),
		}
	default:
		panic(fmt.Errorf("unrecognized order"))
	}
}

func (m Map[K, V, PV]) GetAll(ctx sdk.Context) []V {
	iter := m.Iterate(ctx, Unbounded(), Unbounded(), OrderAscending)
	defer iter.Close()

	var list []V
	for ; iter.Valid(); iter.Next() {
		list = append(list, iter.Value())
	}

	return list
}

type Iterator[K Key, V any, PV interface {
	*V
	Object
}] struct {
	cdc  codec.BinaryCodec
	iter sdk.Iterator
}

func (i Iterator[K, V, PV]) Close() {
	_ = i.iter.Close()
}

func (i Iterator[K, V, PV]) Next() {
	i.iter.Next()
}

func (i Iterator[K, V, PV]) Valid() bool {
	return i.iter.Valid()
}

func (i Iterator[K, V, PV]) Value() V {
	x := PV(new(V))
	i.cdc.MustUnmarshal(i.iter.Value(), x)
	return *x
}

func (i Iterator[K, V, PV]) Key() K {
	var k K
	return k.FromPrimaryKeyBytes(i.iter.Key()).(K) // TODO implement this better
}
