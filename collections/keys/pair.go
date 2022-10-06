package keys

import (
	"fmt"
)

// Join joins the two parts of a Pair key.
func Join[K1 Key, K2 Key](k1 K1, k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		p1: &k1,
		p2: &k2,
	}
}

// PairPrefix is used to provide only the K1 part of the Pair.
// Usually used in Range.Prefix where Key is Pair.
func PairPrefix[K1 Key, K2 Key](k1 K1) Pair[K1, K2] {
	return Pair[K1, K2]{
		p1: &k1,
		p2: nil,
	}
}

// PairSuffix is used to provide only the K2 part of the Pair.
// Usually used in Range.Start or Range.End where Key is Pair.
func PairSuffix[K1 Key, K2 Key](k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		p1: nil,
		p2: &k2,
	}
}

// Pair represents a multipart key composed of
// two Key of different or equal types.
type Pair[K1 Key, K2 Key] struct {
	// p1 is the first part of the Pair.
	p1 *K1
	// p2 is the second part of the Pair.
	p2 *K2
}

// K1 returns the first part of the key,
// if present. If the key is not present
// the zero value is returned.
func (t Pair[K1, K2]) K1() K1 {
	if t.p1 != nil {
		return *t.p1
	} else {
		var x K1
		return x
	}
}

// K2 returns the second part of the key,
// if present, If the key is not present
// the zero value is returned.
func (t Pair[K1, K2]) K2() K2 {
	if t.p2 != nil {
		return *t.p2
	} else {
		var x K2
		return x
	}
}

func (t Pair[K1, K2]) fkb1(b []byte) (int, K1) {
	var k1 K1
	i, p1 := k1.FromKeyBytes(b)
	return i, p1.(K1)
}

func (t Pair[K1, K2]) fkb2(b []byte) (int, K2) {
	var k2 K2
	i, p2 := k2.FromKeyBytes(b)
	return i, p2.(K2)
}

func (t Pair[K1, K2]) FromKeyBytes(b []byte) (int, Key) {
	// NOTE(mercilex): is it always safe to assume that when we get a part
	// of the key it's going to always contain the full key and not only a part?
	i1, k1 := t.fkb1(b)
	i2, k2 := t.fkb2(b[i1:])
	return i1 + i2, Pair[K1, K2]{
		p1: &k1,
		p2: &k2,
	}
}

func (t Pair[K1, K2]) KeyBytes() []byte {
	if t.p1 != nil && t.p2 != nil {
		return append((*t.p1).KeyBytes(), (*t.p2).KeyBytes()...)
	} else if t.p1 != nil && t.p2 == nil {
		return (*t.p1).KeyBytes()
	} else if t.p1 == nil && t.p2 != nil {
		return (*t.p2).KeyBytes()
	} else {
		panic("empty Pair key")
	}
}

func (t Pair[K1, K2]) String() string {
	p1 := "<nil>"
	p2 := "<nil>"
	if t.p1 != nil {
		p1 = (*t.p1).String()
	}
	if t.p2 != nil {
		p2 = (*t.p2).String()
	}

	return fmt.Sprintf("('%s', '%s')", p1, p2)
}

// PairRange implements the Range interface
// to provide an easier way to range over Pair keys.
type PairRange[K1, K2 Key] struct {
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
	bound := Inclusive(start)
	p.start = &bound
	return p
}

func (p PairRange[K1, K2]) StartExclusive(start K2) PairRange[K1, K2] {
	bound := Exclusive(start)
	p.start = &bound
	return p
}

func (p PairRange[K1, K2]) EndInclusive(end K2) PairRange[K1, K2] {
	bound := Inclusive(end)
	p.end = &bound
	return p
}

func (p PairRange[K1, K2]) EndExclusive(end K2) PairRange[K1, K2] {
	bound := Exclusive(end)
	p.end = &bound
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
		prefix = &Pair[K1, K2]{p1: p.prefix}
	}
	if p.start != nil {
		start = &Bound[Pair[K1, K2]]{
			value:     Pair[K1, K2]{p2: &p.start.value},
			inclusive: p.start.inclusive,
		}
	}
	if p.end != nil {
		end = &Bound[Pair[K1, K2]]{
			value:     Pair[K1, K2]{p2: &p.end.value},
			inclusive: p.end.inclusive,
		}
	}

	order = p.order
	return
}
