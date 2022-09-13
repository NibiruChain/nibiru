package keys

// NewRange returns a Range instance
// which iterates over all keys in
// ascending order.
func NewRange[K Key]() Range[K] {
	return Range[K]{
		prefix: nil,
		start:  nil,
		end:    nil,
		order:  OrderAscending,
	}
}

// Range defines a range of keys.
type Range[K Key] struct {
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

func (r Range[K]) Compile() (prefix []byte, start []byte, end []byte, order Order) {
	order = r.order
	if r.prefix != nil {
		prefix = (*r.prefix).KeyBytes()
	}
	if r.start != nil {
		start = r.start.Bytes()
	}
	if r.end != nil {
		end = r.end.Bytes()
	}
	return
}

type Bound[K Key] struct {
	some      []byte
	exclusive bool
}

func (b Bound[K]) Bytes() []byte {
	if b.exclusive {
		return b.some
	} else {
		panic("not impl")
	}
}

// Inclusive creates a key Bound which is inclusive.
func Inclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		some:      k.KeyBytes(),
		exclusive: true,
	}
}

// Exclusive creates a key Bound which is exclusive.
func Exclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		some:      k.KeyBytes(),
		exclusive: false,
	}
}
