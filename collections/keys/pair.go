package keys

import (
	"fmt"
)

// Join joins the two parts of a Pair key.
func Join[K1 Key, K2 Key](k1 K1, k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		P1: k1,
		P2: k2,
	}
}

// Pair represents a multipart key composed of
// two Key of different or equal types.
type Pair[K1 Key, K2 Key] struct {
	// P1 is the first part of the Pair.
	P1 K1
	// P2 is the second part of the Pair.
	P2 K2
}

func (t Pair[K1, K2]) FromKeyBytes(b []byte) (int, Key) {
	i1, k1 := t.P1.FromKeyBytes(b)
	i2, k2 := t.P2.FromKeyBytes(b[i1+1:]) // add one to not pass last index
	// add one back as the indexes reported back will start from the last index + 1
	return i1 + i2 + 1, Pair[K1, K2]{
		P1: k1.(K1),
		P2: k2.(K2),
	}
}

func (t Pair[K1, K2]) KeyBytes() []byte {
	return append(t.P1.KeyBytes(), t.P2.KeyBytes()...)
}

func (t Pair[K1, K2]) String() string {
	return fmt.Sprintf("('%s', '%s')", t.P1.String(), t.P2.String())
}
