package collections

import (
	"errors"
)

// ErrNotFound is returned when an object is not found.
var ErrNotFound = errors.New("collections: not found")

// ErrNilIntKey is returned when encoding a nil [cosmossdk.io/math.Int] as a key or value.
var ErrNilIntKey = errors.New("collections: cannot encode nil math.Int")

// ErrNegativeIntKey is returned when encoding a negative [cosmossdk.io/math.Int] as a
// lexicographic collection key (unsigned padded encoding).
var ErrNegativeIntKey = errors.New("collections: cannot encode negative math.Int")

// ErrEmptyPairKey is returned when encoding an empty Pair (both sides nil).
var ErrEmptyPairKey = errors.New("collections: empty Pair key")

// Namespace defines a storage namespace which must be unique in a single module
// for all the different storage layer types: Map, Sequence, KeySet, Item, MultiIndex, IndexedMap
type Namespace uint8

func (n Namespace) Prefix() []byte { return []byte{uint8(n)} }

// KeyEncoder defines a generic interface which is implemented
// by types that are capable of encoding and decoding collections keys.
type KeyEncoder[T any] interface {
	// Encode encodes the type T into bytes.
	Encode(key T) ([]byte, error)
	// Decode decodes the given bytes back into T.
	// And it also must return the bytes of the buffer which were read.
	Decode(b []byte) (int, T)
	// Stringify returns a string representation of T.
	Stringify(key T) string
}

// ValueEncoder defines a generic interface which is implemented
// by types that are capable of encoding and decoding collection values.
type ValueEncoder[T any] interface {
	// Encode encodes the value T into bytes.
	Encode(value T) ([]byte, error)
	// Decode returns the type T given its bytes representation.
	Decode(b []byte) T
	// Stringify returns a string representation of T.
	Stringify(value T) string
	// Name returns the name of the object.
	Name() string
}
