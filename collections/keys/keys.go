package keys

import (
	"fmt"
)

// Order defines the ordering of keys.
type Order uint8

const (
	OrderAscending Order = iota
	OrderDescending
)

// Key defines a type which can be converted to and from bytes.
// Constraints:
// - It's ordered, meaning, for example:
//		StringKey("a").KeyBytes() < StringKey("b").KeyBytes().
//      Int64Key(100).KeyBytes() > Int64Key(-100).KeyBytes()
// - Going back and forth using KeyBytes and FromKeyBytes produces the same results.
// - It's prefix safe, meaning that bytes.Contains(StringKey("a").KeyBytes(), StringKey("aa").KeyBytes()) = false.
type Key interface {
	// KeyBytes returns the key as bytes.
	KeyBytes() []byte
	// FromKeyBytes parses the Key from bytes.
	// returns i which is the index of the end of the key.
	// Constraint: Key == Self (aka the interface implementer).
	// NOTE(mercilex): we in theory should return Key[T any] and constrain
	// in the collections.Map, collections.IndexedMap, collections.Set
	// that T is in fact the Key itself.
	// We don't do it otherwise all our APIs would get messy
	// due to golang's compiler type inference.
	FromKeyBytes(b []byte) (i int, k Key)
	// Stringer is implemented to allow human-readable formats, especially important in errors.
	fmt.Stringer
}

type Uint8Key uint8

func (u Uint8Key) KeyBytes() []byte {
	return []byte{uint8(u)}
}

func (u Uint8Key) FromKeyBytes(b []byte) (i int, k Key) {
	return 0, Uint8Key(b[0])
}

func (u Uint8Key) String() string { return fmt.Sprintf("%d", u) }

type Uint32 uint32

type Uint64 uint64

type Int64 int64

func validString[T ~string](s T) error {
	for i, c := range s {
		if c == 0 {
			return fmt.Errorf("invalid null character at index %d: %s", i, s)
		}
	}
	return nil
}
