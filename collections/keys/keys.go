package keys

import (
	"fmt"
)

// Order defines the ordering of keys.
type Order uint8

const (
	// OrderAscending defines an order going from the
	// smallest key to the biggest key.
	OrderAscending Order = iota
	// OrderDescending defines an order going from the
	// biggest key to the smallest. In the KVStore
	// it equals to iterating in reverse.
	OrderDescending
)

// Key defines a type which can be converted to and from bytes.
// Constraints:
//   - It's ordered, meaning, for example:
//     StringKey("a").KeyBytes() < StringKey("b").KeyBytes().
//     Int64Key(100).KeyBytes() > Int64Key(-100).KeyBytes()
//   - Going back and forth using KeyBytes and FromKeyBytes produces the same results.
//   - It's prefix safe, meaning that bytes.Contains(StringKey("a").KeyBytes(), StringKey("aa").KeyBytes()) = false.
type Key interface {
	// KeyBytes returns the key as bytes.
	KeyBytes() []byte
	// FromKeyBytes parses the Key from bytes.
	// returns i which is the numbers of bytes read from the buffer.
	// Constraint: Key == Self (aka the interface implementer).
	// NOTE(mercilex): we in theory should return Key[T any] and constrain
	// in the collections.Map, collections.IndexedMap, collections.Set
	// that T is in fact the Key itself.
	// We don't do it otherwise all our APIs would get messy
	// due to golang's compiler type inference.
	FromKeyBytes(buf []byte) (i int, k Key)
	// Stringer is implemented to allow human-readable formats, especially important in errors.
	fmt.Stringer
}

func validString[T ~string](s T) error {
	for i, c := range s {
		if c == 0 {
			return fmt.Errorf("invalid null character at index %d: %s", i, s)
		}
	}
	return nil
}
