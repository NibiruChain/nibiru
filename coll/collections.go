package coll

import (
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"time"
)

var ErrNotFound = errors.New("collections: not found")

type keys struct {
	String     KeyEncoder[string]
	AccAddress KeyEncoder[sdk.AccAddress]
	Time       KeyEncoder[time.Time]
	Uint64     KeyEncoder[uint64]
}

// Keys is a helper struct that groups together all available key encoder types.
var Keys = keys{
	String:     stringKey{},
	AccAddress: accAddressKey{},
	Time:       timeKey{},
	Uint64:     uint64Key{},
}

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

type uint64Key struct{}

func (uint64Key) Stringify(u uint64) string        { return strconv.FormatUint(u, 10) }
func (uint64Key) KeyEncode(u uint64) []byte        { return sdk.Uint64ToBigEndian(u) }
func (uint64Key) KeyDecode(b []byte) (int, uint64) { return 8, sdk.BigEndianToUint64(b) }

type timeKey struct{}

func (timeKey) Stringify(t time.Time) string { return t.String() }
func (timeKey) KeyEncode(t time.Time) []byte { return Keys.Uint64.KeyEncode(uint64(t.UnixMilli())) }
func (timeKey) KeyDecode(b []byte) (int, time.Time) {
	i, u := Keys.Uint64.KeyDecode(b)
	return i, time.UnixMilli(int64(u))
}

type accAddressKey struct{}

func (accAddressKey) Stringify(addr sdk.AccAddress) string { return addr.String() }
func (accAddressKey) KeyEncode(addr sdk.AccAddress) []byte {
	return Keys.String.KeyEncode(addr.String())
}
func (accAddressKey) KeyDecode(b []byte) (int, sdk.AccAddress) {
	i, s := Keys.String.KeyDecode(b)
	return i, sdk.MustAccAddressFromBech32(s)
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
