package coll

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Order uint8

const OrderAscending = 0
const OrderDescending = 1

type Bound[K any] struct {
	value     K
	inclusive bool
}

type Ranger[K any] interface {
	RangeValues() (prefix *K, start *Bound[K], end *Bound[K], order Order)
}

// Range is a Ranger implementer.
type Range[K any] struct {
	prefix *K
	start  *Bound[K]
	end    *Bound[K]
	order  Order
}

// Prefix sets a fixed prefix for the key range.
func (r Range[K]) Prefix(key K) Range[K] {
	r.prefix = &key
	return r
}

// Start sets the start range of the key.
func (r Range[K]) Start(bound Bound[K]) Range[K] {
	r.start = &bound
	return r
}

// End sets the end range of the key.
func (r Range[K]) End(bound Bound[K]) Range[K] {
	r.end = &bound
	return r
}

// Descending sets the key range to be inverse.
func (r Range[K]) Descending() Range[K] {
	r.order = OrderDescending
	return r
}

func (r Range[K]) RangeValues() (prefix *K, start *Bound[K], end *Bound[K], order Order) {
	return r.prefix, r.start, r.end, r.order
}

func iteratorFromRange[K, V any](s sdk.KVStore, r Ranger[K], kc KeyEncoder[K], vc ValueEncoder[V]) Iterator[K, V] {
	pfx, start, end, order := r.RangeValues()
	var prefixBytes []byte
	if pfx != nil {
		prefixBytes = kc.KeyEncode(*pfx)
		s = prefix.NewStore(s, prefixBytes)
	}
	var startBytes []byte // default is nil
	if start != nil {
		startBytes = kc.KeyEncode(start.value)
		// iterators are inclusive at start by default
		// so if we want to make the iteration exclusive
		// we extend by one byte.
		if !start.inclusive {
			startBytes = extendOneByte(startBytes)
		}
	}
	var endBytes []byte // default is nil
	if end != nil {
		endBytes = kc.KeyEncode(end.value)
		// iterators are exclusive at end by default
		// so if we want to make the iteration
		// inclusive we need to extend by one byte.
		if end.inclusive {
			endBytes = extendOneByte(endBytes)
		}
	}

	var iter sdk.Iterator
	switch order {
	case OrderAscending:
		iter = s.Iterator(startBytes, endBytes)
	case OrderDescending:
		iter = s.ReverseIterator(startBytes, endBytes)
	default:
		panic("unrecognized order")
	}

	return Iterator[K, V]{
		kc:          kc,
		vc:          vc,
		iter:        iter,
		prefixBytes: prefixBytes,
	}
}

type Iterator[K, V any] struct {
	kc KeyEncoder[K]
	vc ValueEncoder[V]

	iter sdk.Iterator

	prefixBytes []byte
}

func (i Iterator[K, V]) Close() { _ = i.iter.Close() }

func (i Iterator[K, V]) Next() { i.iter.Next() }

func (i Iterator[K, V]) Valid() bool { return i.iter.Valid() }

func (i Iterator[K, V]) Value() V {
	return i.vc.ValueDecode(i.iter.Value())
}

func (i Iterator[K, V]) Key() K {
	rawKey := append(i.prefixBytes, i.iter.Key()...)
	_, c := i.kc.KeyDecode(rawKey) // todo(mercilex): can we assert safety here?
	return c
}

// TODO doc
func (i Iterator[K, V]) Values() []V {
	defer i.Close()

	var values []V
	for ; i.iter.Valid(); i.iter.Next() {
		values = append(values, i.Value())
	}
	return values
}

// TODO doc
func (i Iterator[K, V]) Keys() []K {
	defer i.Close()

	var keys []K
	for ; i.iter.Valid(); i.iter.Next() {
		keys = append(keys, i.Key())
	}
	return keys
}

// todo doc
func (i Iterator[K, V]) KeyValues() []KeyValue[K, V] {
	defer i.Close()

	var kvs []KeyValue[K, V]
	for ; i.iter.Valid(); i.iter.Next() {
		kvs = append(kvs, KeyValue[K, V]{
			Key:   i.Key(),
			Value: i.Value(),
		})
	}

	return kvs
}

type KeyValue[K, V any] struct {
	Key   K
	Value V
}

func extendOneByte(b []byte) []byte {
	return append(b, 0)
}
