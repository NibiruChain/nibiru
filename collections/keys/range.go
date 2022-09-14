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
		start = r.compileStart()
	}
	if r.end != nil {
		end = r.compileEnd()
	}
	return
}

func (r Range[K]) compileStart() []byte {
	bytes := r.start.value.KeyBytes()
	// iterator start is inclusive by default
	if r.start.inclusive {
		return bytes
	} else {
		return extendOneByte(bytes)
	}
}

func (r Range[K]) compileEnd() []byte {
	bytes := r.end.value.KeyBytes()
	// iterator end is exclusive by default
	if !r.end.inclusive {
		return bytes
	} else {
		return extendOneByte(bytes)
	}
}

func extendOneByte(b []byte) []byte {
	return append(b, 0)
}

type Bound[K Key] struct {
	value     K
	inclusive bool
}

// Inclusive creates a key Bound which is inclusive.
func Inclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		value:     k,
		inclusive: true,
	}
}

// Exclusive creates a key Bound which is exclusive.
func Exclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		value:     k,
		inclusive: false,
	}
}
