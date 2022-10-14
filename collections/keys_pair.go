package collections

import "strings"

// PairKeyEncoder creates a new KeyEncoder for Pair types, give the two key encoders for K1 and K2.
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

// Stringify returns a string representation of the given Pair.
func (p pairKeyEncoder[K1, K2]) Stringify(key Pair[K1, K2]) string {
	s := strings.Builder{}
	s.WriteByte('(')
	if key.k1 == nil {
		s.WriteString("<nil>")
	} else {
		s.WriteByte('"')
		s.WriteString(p.kc1.Stringify(*key.k1))
		s.WriteByte('"')
	}
	s.WriteByte(',')
	s.WriteByte(' ')
	if key.k2 == nil {
		s.WriteString("<nil>")
	} else {
		s.WriteByte('"')
		s.WriteString(p.kc2.Stringify(*key.k2))
		s.WriteByte('"')
	}
	s.WriteByte(')')
	return s.String()
}

// Encode encodes the Pair.
// If both parts of the keys are present, then the byte version of K1 and K2 are joined together.
// If only the first part is present then
func (p pairKeyEncoder[K1, K2]) Encode(key Pair[K1, K2]) []byte {
	if key.k1 != nil && key.k2 != nil {
		return append(p.kc1.Encode(*key.k1), p.kc2.Encode(*key.k2)...)
	} else if key.k1 != nil && key.k2 == nil {
		return p.kc1.Encode(*key.k1)
	} else if key.k1 == nil && key.k2 != nil {
		return p.kc2.Encode(*key.k2)
	} else {
		panic("empty Pair key")
	}
}

// Decode decodes the Pair. It assumes that the provided bytes contain both the K1 and K2 part.
func (p pairKeyEncoder[K1, K2]) Decode(b []byte) (int, Pair[K1, K2]) {
	// NOTE(mercilex): is it always safe to assume that when we get a part
	// of the key it's going to always contain the full key and not only a part?
	i1, k1 := p.kc1.Decode(b)
	i2, k2 := p.kc2.Decode(b[i1:])
	return i1 + i2, Pair[K1, K2]{
		k1: &k1,
		k2: &k2,
	}
}

// Join returns a fully populated Pair
// given the two key parts.
func Join[K1, K2 any](k1 K1, k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		k1: &k1,
		k2: &k2,
	}
}

// PairPrefix returns a partially populated pair
// given the first part of the key.
func PairPrefix[K1, K2 any](k1 K1) Pair[K1, K2] {
	return Pair[K1, K2]{
		k1: &k1,
	}
}

// PairSuffix returns a partially populated pair
// given the last part of the key.
func PairSuffix[K1, K2 any](k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		k2: &k2,
	}
}

// Pair defines a storage key composed of two keys of different or equal types.
type Pair[K1, K2 any] struct {
	k1 *K1
	k2 *K2
}

func (p Pair[K1, K2]) K2() (k2 K2) {
	if p.k2 != nil {
		k2 = *p.k2
	}
	return
}

func (p Pair[K1, K2]) K1() (k1 K1) {
	if p.k1 != nil {
		k1 = *p.k1
	}
	return
}

// PairRange implements the Ranger interface
// to provide an easier way to range over Pair keys.
type PairRange[K1, K2 any] struct {
	prefix *K1
	start  *Bound[K2]
	end    *Bound[K2]
	order  Order
}

// Prefix makes the range contain only keys starting with the given k1 prefix.
func (p PairRange[K1, K2]) Prefix(prefix K1) PairRange[K1, K2] {
	p.prefix = &prefix
	return p
}

// StartInclusive makes the range contain only keys which are bigger or equal to the provided start K2.
func (p PairRange[K1, K2]) StartInclusive(start K2) PairRange[K1, K2] {
	p.start = BoundInclusive(start)
	return p
}

// StartExclusive makes the range contain only keys which are bigger to the provided start K2.
func (p PairRange[K1, K2]) StartExclusive(start K2) PairRange[K1, K2] {
	p.start = BoundExclusive(start)
	return p
}

// EndInclusive makes the range contain only keys which are smaller or equal to the provided end K2.
func (p PairRange[K1, K2]) EndInclusive(end K2) PairRange[K1, K2] {
	p.end = BoundInclusive(end)
	return p
}

// EndExclusive makes the range contain only keys which are smaller to the provided end K2.
func (p PairRange[K1, K2]) EndExclusive(end K2) PairRange[K1, K2] {
	p.end = BoundExclusive(end)
	return p
}

// Descending makes the range run in reverse (bigger->smaller, instead of smaller->bigger)
func (p PairRange[K1, K2]) Descending() PairRange[K1, K2] {
	p.order = OrderDescending
	return p
}

// RangeValues implements Ranger for Pair[K1, K2].
// If start and end are set, prefix must be set too or the function call will panic.
// The implementation returns a range which prefixes over the K1 prefix.
// And the key range goes from K2 start to K2 end (if any are defined).
// Example:
// given the following keys in storage:
// Pair["milan", "person1"]
// Pair["milan", "person2"]
// Pair["milan", "person3"]
// Pair["new york", "person0"]
// doing: PairRange[string, string].Prefix("milan").StartInclusive("person1").EndExclusive("person3")
// returns: Pair["milan", "person1"], Pair["milan", "person2"]
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
