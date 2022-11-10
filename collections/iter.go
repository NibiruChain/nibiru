package collections

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Order defines the key order.
type Order uint8

const (
	// OrderAscending instructs the Iterator to provide keys from the smallest to the greatest.
	OrderAscending Order = 0
	// OrderDescending instructs the Iterator to provide keys from the greatest to the smallest.
	OrderDescending Order = 1
)

// BoundInclusive creates a Bound of the provided key K
// which is inclusive. Meaning, if it is used as Ranger.RangeValues start,
// the provided key will be included if it exists in the Iterator range.
func BoundInclusive[K any](key K) *Bound[K] {
	return &Bound[K]{
		value:     key,
		inclusive: true,
	}
}

// BoundExclusive creates a Bound of the provided key K
// which is exclusive. Meaning, if it is used as Ranger.RangeValues start,
// the provided key will be excluded if it exists in the Iterator range.
func BoundExclusive[K any](key K) *Bound[K] {
	return &Bound[K]{
		value:     key,
		inclusive: false,
	}
}

// Bound defines key bounds for Start and Ends of iterator ranges.
type Bound[K any] struct {
	value     K
	inclusive bool
}

// Ranger defines a generic interface that provides a range of keys.
type Ranger[K any] interface {
	// RangeValues is defined by Ranger implementers.
	// It provides instructions to generate an Iterator instance.
	// If prefix is not nil, then the Iterator will return only the keys which start
	// with the given prefix.
	// If start is not nil, then the Iterator will return only keys which are greater than the provided start
	// or greater equal depending on the bound is inclusive or exclusive.
	// If end is not nil, then the Iterator will return only keys which are smaller than the provided end
	// or smaller equal depending on the bound is inclusive or exclusive.
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

// StartInclusive makes the range contain only keys which are bigger or equal to the provided start K.
func (r Range[K]) StartInclusive(start K) Range[K] {
	r.start = BoundInclusive(start)
	return r
}

// StartExclusive makes the range contain only keys which are bigger to the provided start K.
func (r Range[K]) StartExclusive(start K) Range[K] {
	r.start = BoundExclusive(start)
	return r
}

// EndInclusive makes the range contain only keys which are smaller or equal to the provided end K.
func (r Range[K]) EndInclusive(end K) Range[K] {
	r.end = BoundInclusive(end)
	return r
}

// EndExclusive makes the range contain only keys which are smaller to the provided end K.
func (r Range[K]) EndExclusive(end K) Range[K] {
	r.end = BoundExclusive(end)
	return r
}

func (r Range[K]) Descending() Range[K] {
	r.order = OrderDescending
	return r
}

func (r Range[K]) RangeValues() (prefix *K, start *Bound[K], end *Bound[K], order Order) {
	return r.prefix, r.start, r.end, r.order
}

// iteratorFromRange generates an Iterator instance, with the proper prefixing and ranging.
func iteratorFromRange[K, V any](s sdk.KVStore, r Ranger[K], kc KeyEncoder[K], vc ValueEncoder[V]) Iterator[K, V] {
	pfx, start, end, order := r.RangeValues()
	var prefixBytes []byte
	if pfx != nil {
		prefixBytes = kc.Encode(*pfx)
		s = prefix.NewStore(s, prefixBytes)
	}
	var startBytes []byte // default is nil
	if start != nil {
		startBytes = kc.Encode(start.value)
		// iterators are inclusive at start by default
		// so if we want to make the iteration exclusive
		// we extend by one byte.
		if !start.inclusive {
			startBytes = extendOneByte(startBytes)
		}
	}
	var endBytes []byte // default is nil
	if end != nil {
		endBytes = kc.Encode(end.value)
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
		panic("unrecognized Order")
	}

	return Iterator[K, V]{
		kc:          kc,
		vc:          vc,
		iter:        iter,
		prefixBytes: prefixBytes,
	}
}

// Iterator defines a generic wrapper around an sdk.Iterator.
// This iterator provides automatic key and value encoding,
// it assumes all the keys and values contained within the sdk.Iterator
// range are the same.
type Iterator[K, V any] struct {
	kc KeyEncoder[K]
	vc ValueEncoder[V]

	iter sdk.Iterator

	prefixBytes []byte
}

// Value returns the current iterator value bytes decoded.
func (i Iterator[K, V]) Value() V {
	return i.vc.Decode(i.iter.Value())
}

// Key returns the current sdk.Iterator decoded key.
func (i Iterator[K, V]) Key() K {
	rawKey := append(i.prefixBytes, i.iter.Key()...)
	read, c := i.kc.Decode(rawKey)
	if read != len(rawKey) {
		panic(fmt.Sprintf("key decoder didn't fully consume the key: %T %x %d", i.kc, rawKey, read))
	}
	return c
}

// Values fully consumes the iterator and returns all the decoded values contained within the range.
func (i Iterator[K, V]) Values() []V {
	defer i.Close()

	var values []V
	for ; i.iter.Valid(); i.iter.Next() {
		values = append(values, i.Value())
	}
	return values
}

// Keys fully consumes the iterator and returns all the decoded keys contained within the range.
func (i Iterator[K, V]) Keys() []K {
	defer i.Close()

	var keys []K
	for ; i.iter.Valid(); i.iter.Next() {
		keys = append(keys, i.Key())
	}
	return keys
}

// KeyValue returns the current key and value decoded.
func (i Iterator[K, V]) KeyValue() KeyValue[K, V] {
	return KeyValue[K, V]{
		Key:   i.Key(),
		Value: i.Value(),
	}
}

// KeyValues fully consumes the iterator and returns the list of key and values within the iterator range.
func (i Iterator[K, V]) KeyValues() []KeyValue[K, V] {
	defer i.Close()

	var kvs []KeyValue[K, V]
	for ; i.iter.Valid(); i.iter.Next() {
		kvs = append(kvs, i.KeyValue())
	}

	return kvs
}

func (i Iterator[K, V]) Close()      { _ = i.iter.Close() }
func (i Iterator[K, V]) Next()       { i.iter.Next() }
func (i Iterator[K, V]) Valid() bool { return i.iter.Valid() }

type KeyValue[K, V any] struct {
	Key   K
	Value V
}

func extendOneByte(b []byte) []byte {
	return append(b, 0)
}
