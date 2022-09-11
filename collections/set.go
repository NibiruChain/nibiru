package collections

import (
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Set[K keys.Key] struct {
	cdc    codec.BinaryCodec
	sk     sdk.StoreKey
	prefix []byte
	_      K
}

func NewSet[K keys.Key](cdc codec.BinaryCodec, sk sdk.StoreKey, prefix uint8) Set[K] {
	return Set[K]{
		cdc:    cdc,
		sk:     sk,
		prefix: []byte{prefix},
	}
}

func (s Set[K]) Has(ctx sdk.Context, k K) bool {
	return s.getStore(ctx).Has(k.PrimaryKey())
}

func (s Set[K]) Insert(ctx sdk.Context, k K) {
	s.getStore(ctx).Set(k.PrimaryKey(), []byte{})
}

func (s Set[K]) Delete(ctx sdk.Context, k K) {
	s.getStore(ctx).Delete(k.PrimaryKey())
}

func (s Set[K]) Iterate(ctx sdk.Context, start, end keys.Bound[K], order keys.Order) SetIterator[K] {
	store := s.getStore(ctx)
	return SetIterator[K]{
		iter: newMapIterator[K, noOpObject](s.cdc, store, start, end, order),
	}
}

func (s Set[K]) GetAll(ctx sdk.Context) []K {
	iter := s.Iterate(ctx, keys.None[K](), keys.None[K](), keys.OrderAscending)
	defer iter.Close()

	var k []K
	for ; iter.Valid(); iter.Next() {
		k = append(k, iter.Key())
	}
	return k
}

func (s Set[K]) getStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(s.sk), s.prefix)
}

type SetIterator[K keys.Key] struct {
	iter MapIterator[K, noOpObject, *noOpObject]
}

func (s SetIterator[K]) Close() {
	s.iter.Close()
}

func (s SetIterator[K]) Next() {
	s.iter.Next()
}

func (s SetIterator[K]) Valid() bool {
	return s.iter.Valid()
}

func (s SetIterator[K]) Key() K {
	return s.iter.Key()
}
