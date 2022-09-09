package collections

type Bound struct {
	some      []byte
	inclusive bool
	isNone    bool
}

func (b Bound) bytes() []byte {
	if b.isNone {
		return nil
	} else if b.inclusive {
		return b.some
	} else {
		panic("not impl")
	}
}

func BoundInclusive[K Key](k K) Bound {
	return Bound{
		some:      k.PrimaryKey(),
		inclusive: true,
		isNone:    false,
	}
}

func BoundExclusive[K Key](k K) Bound {
	return Bound{
		some:      k.PrimaryKey(),
		inclusive: false,
		isNone:    false,
	}
}

func Unbounded() Bound {
	return Bound{isNone: true}
}
