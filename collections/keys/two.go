package keys

import "fmt"

// Join joins the two parts of a Pair key.
func Join[K1 Key, K2 Key](k1 K1, k2 K2) Pair[K1, K2] {
	return Pair[K1, K2]{
		k1: k1,
		k2: k2,
	}
}

// Pair represents a multipart key composed of
// two Key of different or equal types.
type Pair[K1 Key, K2 Key] struct {
	k1 K1
	k2 K2
}

func (t Pair[K1, K2]) FromPrimaryKeyBytes(b []byte) Key {
	panic("impl")
}

func (t Pair[K1, K2]) PrimaryKey() []byte {
	return append(t.k1.PrimaryKey(), t.k2.PrimaryKey()...)
}

func (t Pair[K1, K2]) String() string {
	return fmt.Sprintf("('%s', '%s')", t.k1.String(), t.k2.String())
}
