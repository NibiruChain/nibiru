package keys

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IteratorFromRange returns a sdk.Iterator given the range
// and the sdk.KVStore to apply the iterator to.
// prefixBytes MUST not be mutated.
func IteratorFromRange[K Key](store sdk.KVStore, r Range[K]) (iter sdk.Iterator, prefixBytes []byte) {
	pfx, start, end, order := r.RangeValues()
	if pfx != nil {
		prefixBytes = (*pfx).KeyBytes()
		store = prefix.NewStore(store, prefixBytes)
	}
	var startBytes []byte // default is nil
	if start != nil {
		startBytes = start.value.KeyBytes()
		// iterators are inclusive at start by default
		// so if we want to make the iteration exclusive
		// we extend by one byte.
		if !start.inclusive {
			startBytes = extendOneByte(startBytes)
		}
	}
	var endBytes []byte // default is nil
	if end != nil {
		endBytes = end.value.KeyBytes()
		// iterators are exclusive at end by default
		// so if we want to make the iteration
		// inclusive we need to extend by one byte.
		if end.inclusive {
			endBytes = extendOneByte(endBytes)
		}
	}

	switch order {
	case OrderAscending:
		return store.Iterator(startBytes, endBytes), prefixBytes
	case OrderDescending:
		return store.ReverseIterator(startBytes, endBytes), prefixBytes
	default:
		panic("unrecognized order")
	}
}

// Range defines an interface which instructs on how to iterate
// over keys.
type Range[K Key] interface {
	// RangeValues returns the range instructions.
	RangeValues() (prefix *K, start *Bound[K], end *Bound[K], order Order)
}

// NewRange returns a Range instance
// which iterates over all keys in
// ascending order.
func NewRange[K Key]() RawRange[K] {
	return RawRange[K]{
		prefix: nil,
		start:  nil,
		end:    nil,
		order:  OrderAscending,
	}
}

// RawRange is a Range implementer.
type RawRange[K Key] struct {
	prefix *K
	start  *Bound[K]
	end    *Bound[K]
	order  Order
}

// Prefix sets a fixed prefix for the key range.
func (r RawRange[K]) Prefix(key K) RawRange[K] {
	r.prefix = &key
	return r
}

// Start sets the start range of the key.
func (r RawRange[K]) Start(bound Bound[K]) RawRange[K] {
	r.start = &bound
	return r
}

// End sets the end range of the key.
func (r RawRange[K]) End(bound Bound[K]) RawRange[K] {
	r.end = &bound
	return r
}

// Descending sets the key range to be inverse.
func (r RawRange[K]) Descending() RawRange[K] {
	r.order = OrderDescending
	return r
}

func (r RawRange[K]) RangeValues() (prefix *K, start *Bound[K], end *Bound[K], order Order) {
	return r.prefix, r.start, r.end, r.order
}

func extendOneByte(b []byte) []byte {
	return append(b, 0)
}
