package coll

func PairKeyEncoder[K1, K2 any](kc1 KeyEncoder[K1], kc2 KeyEncoder[K2]) KeyEncoder[Pair[K1, K2]] {
	return pairKeyEncoder[K1, K2]{
		kc1: kc1,
		kc2: kc2,
	}
}

type pairKeyEncoder[K1, K2 any] struct {
	kc1 KeyEncoder[K1]
	kc2 KeyEncoder[K2]
}

func (p pairKeyEncoder[K1, K2]) Stringify(key Pair[K1, K2]) string {
	k1, k2 := "<nil>", "<nil>"
	if key.k1 != nil {
		k1 = p.kc1.Stringify(*key.k1)
	}
	if key.k2 != nil {
		k2 = p.kc2.Stringify(*key.k2)
	}
	return "('" + k1 + "', '" + k2 + "')"
}

func (p pairKeyEncoder[K1, K2]) KeyEncode(key Pair[K1, K2]) []byte {
	if key.k1 != nil && key.k2 != nil {
		return append(p.kc1.KeyEncode(*key.k1), p.kc2.KeyEncode(*key.k2)...)
	} else if key.k1 != nil && key.k2 == nil {
		return p.kc1.KeyEncode(*key.k1)
	} else if key.k1 == nil && key.k2 != nil {
		return p.kc2.KeyEncode(*key.k2)
	} else {
		panic("empty Pair key")
	}
}

func (p pairKeyEncoder[K1, K2]) KeyDecode(b []byte) (int, Pair[K1, K2]) {
	// NOTE(mercilex): is it always safe to assume that when we get a part
	// of the key it's going to always contain the full key and not only a part?
	i1, k1 := p.kc1.KeyDecode(b)
	i2, k2 := p.kc2.KeyDecode(b[i1:])
	return i1 + i2, Pair[K1, K2]{
		k1: &k1,
		k2: &k2,
	}
}

func Join[K1, K2 any](k1 K1, k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		k1: &k1,
		k2: &k2,
	}
}

type Pair[K1, K2 any] struct {
	k1 *K1
	k2 *K2
}

// PairRange implements the Ranger interface
// to provide an easier way to range over Pair keys.
type PairRange[K1, K2 any] struct {
	prefix *K1
	start  *Bound[K2]
	end    *Bound[K2]
	order  Order
}

func (p PairRange[K1, K2]) Prefix(prefix K1) PairRange[K1, K2] {
	p.prefix = &prefix
	return p
}

func (p PairRange[K1, K2]) StartInclusive(start K2) PairRange[K1, K2] {
	p.start = &Bound[K2]{
		value:     start,
		inclusive: true,
	}
	return p
}

func (p PairRange[K1, K2]) StartExclusive(start K2) PairRange[K1, K2] {
	p.start = &Bound[K2]{
		value:     start,
		inclusive: false,
	}
	return p
}

func (p PairRange[K1, K2]) EndInclusive(end K2) PairRange[K1, K2] {
	p.end = &Bound[K2]{
		value:     end,
		inclusive: true,
	}
	return p
}

func (p PairRange[K1, K2]) EndExclusive(end K2) PairRange[K1, K2] {
	p.end = &Bound[K2]{
		value:     end,
		inclusive: false,
	}
	return p
}

func (p PairRange[K1, K2]) Descending() PairRange[K1, K2] {
	p.order = OrderDescending
	return p
}

func (p PairRange[K1, K2]) RangeValues() (prefix *Pair[K1, K2], start *Bound[Pair[K1, K2]], end *Bound[Pair[K1, K2]], order Order) {
	if (p.end != nil || p.start != nil) && p.prefix == nil {
		panic("invalid PairRange usage: if end or start are set, prefix must be set too")
	}
	if p.prefix != nil {
		prefix = &Pair[K1, K2]{k1: p.prefix}
	}
	if p.start != nil {
		start = &Bound[Pair[K1, K2]]{
			value:     Pair[K1, K2]{k2: &p.start.value},
			inclusive: p.start.inclusive,
		}
	}
	if p.end != nil {
		end = &Bound[Pair[K1, K2]]{
			value:     Pair[K1, K2]{k2: &p.end.value},
			inclusive: p.end.inclusive,
		}
	}

	order = p.order
	return
}
