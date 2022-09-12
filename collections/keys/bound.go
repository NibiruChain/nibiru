package keys

// None is used when there is no Bound
// in the key used for iteration.
func None[K Key](...K) Bound[K] {
	return Bound[K]{isNone: true}
}

type Bound[k Key] struct {
	some      []byte
	inclusive bool
	isNone    bool
}

func (b Bound[K]) Bytes() []byte {
	if b.isNone {
		return nil
	} else if b.inclusive {
		return b.some
	} else {
		panic("not impl")
	}
}

// Inclusive creates a key Bound which is inclusive.
func Inclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		some:      k.KeyBytes(),
		inclusive: true,
		isNone:    false,
	}
}

// Exclusive creates a key Bound which is exclusive.
func Exclusive[K Key](k K) Bound[K] {
	return Bound[K]{
		some:      k.KeyBytes(),
		inclusive: false,
		isNone:    false,
	}
}
