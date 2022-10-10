package collections

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/codec"
)

// ErrNotFound is returned when an object is not found.
var ErrNotFound = errors.New("collections: not found")

// KeyEncoder defines a generic interface which is implemented
// by types that are capable of encoding and decoding collections keys.
type KeyEncoder[T any] interface {
	// KeyEncode encodes the type T into bytes.
	KeyEncode(key T) []byte
	// KeyDecode decodes the given bytes back into T.
	// And it also must return the bytes of the buffer which were read.
	KeyDecode(b []byte) (int, T)
	// Stringify returns a string representation of T.
	Stringify(key T) string
}

// ValueEncoder defines a generic interface which is implemented
// by types that are capable of encoding and decoding collection values.
type ValueEncoder[T any] interface {
	// ValueEncode encodes the value T into bytes.
	ValueEncode(value T) []byte
	// ValueDecode returns the type T given its bytes representation.
	ValueDecode(b []byte) T
	// Stringify returns a string representation of T.
	Stringify(value T) string
	// Name returns the name of the object.
	Name() string
}

// ProtoValueEncoder returns a protobuf value encoder given the codec.BinaryCodec.
// It's used to convert a specific protobuf object into bytes representation and convert
// the protobuf object bytes representation into the concrete object.
func ProtoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}](cdc codec.BinaryCodec) ValueEncoder[V] {
	return protoValueEncoder[V, PV]{
		cdc: cdc,
	}
}

type protoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}] struct {
	cdc codec.BinaryCodec
}

func (p protoValueEncoder[V, PV]) Name() string               { return proto.MessageName(PV(new(V))) }
func (p protoValueEncoder[V, PV]) ValueEncode(value V) []byte { return p.cdc.MustMarshal(PV(&value)) }
func (p protoValueEncoder[V, PV]) Stringify(v V) string       { return PV(&v).String() }
func (p protoValueEncoder[V, PV]) ValueDecode(b []byte) V {
	v := PV(new(V))
	p.cdc.MustUnmarshal(b, v)
	return *v
}

func validString[T ~string](s T) error {
	for i, c := range s {
		if c == 0 {
			return fmt.Errorf("invalid null character at index %d: %s", i, s)
		}
	}
	return nil
}
