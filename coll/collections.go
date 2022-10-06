package coll

import (
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
)

var ErrNotFound = errors.New("collections: not found")

type keys struct {
	String KeyEncoder[string]
}

var Keys = keys{String: stringKey{}}

type stringKey struct{}

func (stringKey) KeyEncode(s string) []byte {
	if err := validString(s); err != nil {
		panic(fmt.Errorf("invalid StringKey: %w", err))
	}
	return append([]byte(s), 0) // null terminate it for safe prefixing
}

func (stringKey) KeyDecode(b []byte) (int, string) {
	l := len(b)
	if l < 2 {
		panic("invalid StringKey bytes")
	}
	for i, c := range b {
		if c == 0 {
			return i + 1, string(b[:i])
		}
	}
	panic(fmt.Errorf("string is not null terminated: %s", b))
}

func (stringKey) Stringify(s string) string {
	return s
}

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

type ProtoCodec[V any, PV interface {
	*V
	codec.ProtoMarshaler
}] struct {
	cdc codec.BinaryCodec
}

func NewProtoCodec[V any, PV interface {
	*V
	codec.ProtoMarshaler
}](cdc codec.BinaryCodec) ProtoCodec[V, PV] {
	return ProtoCodec[V, PV]{
		cdc: cdc,
	}
}

func (p ProtoCodec[V, PV]) ValueEncode(value V) []byte { return p.cdc.MustMarshal(PV(&value)) }
func (p ProtoCodec[V, PV]) Stringify(v V) string       { return PV(&v).String() }
func (p ProtoCodec[V, PV]) ValueDecode(b []byte) V {
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
