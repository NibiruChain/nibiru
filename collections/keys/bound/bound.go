package bound

import "github.com/NibiruChain/nibiru/collections/keys"

// None is used when there is no Bound
// in the key used for iteration.
var None = Bound{
	isNone: true,
}

type Bound struct {
	some      []byte
	inclusive bool
	isNone    bool
}

func (b Bound) Bytes() []byte {
	if b.isNone {
		return nil
	} else if b.inclusive {
		return b.some
	} else {
		panic("not impl")
	}
}

// Inclusive creates a key Bound which is inclusive.
func Inclusive[K keys.Key](k K) Bound {
	return Bound{
		some:      k.PrimaryKey(),
		inclusive: true,
		isNone:    false,
	}
}

// Exclusive creates a key Bound which is exclusive.
func Exclusive[K keys.Key](k K) Bound {
	return Bound{
		some:      k.PrimaryKey(),
		inclusive: false,
		isNone:    false,
	}
}
