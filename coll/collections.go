package coll

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
)

var ErrNotFound = errors.New("collections: not found")

type KeyEncoder[T any] interface {
	KeyEncode(key T) []byte
	KeyDecode(b []byte) (int, T)
	Stringify(key T) string
}

type ValueEncoder[T any] interface {
	ValueEncode(value T) []byte
	ValueDecode(b []byte) T
	Stringify(value T) string
}

type protoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}] struct {
	cdc codec.BinaryCodec
}

func ProtoValueEncoder[V any, PV interface {
	*V
	codec.ProtoMarshaler
}](cdc codec.BinaryCodec) ValueEncoder[V] {
	return protoValueEncoder[V, PV]{
		cdc: cdc,
	}
}

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
