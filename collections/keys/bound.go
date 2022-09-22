package keys

// Bound defines a key bound.
type Bound[K Key] struct {
	value     K    // value is the concrete key.
	inclusive bool // inclusive defines if the key bound should include or not the provided value.
}

// Inclusive creates a key Bound which is inclusive,
// which means the provided key will be included
// in the key range (if present).
func Inclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		value:     k,
		inclusive: true,
	}
}

// Exclusive creates a key Bound which is exclusive,
// which means the provided key will be excluded from
// the key range.
func Exclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		value:     k,
		inclusive: false,
	}
}
