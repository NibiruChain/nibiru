package collections

import (
	"errors"
)

// ErrNotFound is returned when an object is not found.
var ErrNotFound = errors.New("collections: not found")

// Namespace defines a storage namespace which must be unique in a single module
// for all the different storage layer types: Map, Sequence, KeySet, Item, MultiIndex, IndexedMap
type Namespace uint8

func (n Namespace) Prefix() []byte { return []byte{uint8(n)} }

// KeyEncoder defines a generic interface which is implemented
// by types that are capable of encoding and decoding collections keys.
type KeyEncoder[T any] interface {
	// Encode encodes the type T into bytes.
	Encode(key T) []byte
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
	Encode(value T) []byte
	// Decode returns the type T given its bytes representation.
	Decode(b []byte) T
	// Stringify returns a string representation of T.
	Stringify(value T) string
	// Name returns the name of the object.
	Name() string
}
