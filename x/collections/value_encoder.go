package collections

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

var (
	AccAddressValueEncoder ValueEncoder[sdk.AccAddress] = accAddressValueEncoder{}
	DecValueEncoder        ValueEncoder[sdk.Dec]        = decValueEncoder{}
	IntValueEncoder        ValueEncoder[sdkmath.Int]    = intValueEncoder{}
	UintValueEncoder       ValueEncoder[sdkmath.Uint]   = uintValueEncoder{}
	Uint64ValueEncoder     ValueEncoder[uint64]         = uint64Value{}
)

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

func (p protoValueEncoder[V, PV]) Name() string          { return proto.MessageName(PV(new(V))) }
func (p protoValueEncoder[V, PV]) Encode(value V) []byte { return p.cdc.MustMarshal(PV(&value)) }
func (p protoValueEncoder[V, PV]) Stringify(v V) string  { return PV(&v).String() }
func (p protoValueEncoder[V, PV]) Decode(b []byte) V {
	v := PV(new(V))
	p.cdc.MustUnmarshal(b, v)
	return *v
}

// DecValueEncoder ValueEncoder[sdk.Dec]

type decValueEncoder struct{}

func (d decValueEncoder) Encode(value sdk.Dec) []byte {
	b, err := value.Marshal()
	if err != nil {
		panic(fmt.Errorf("%w %s", err, HumanizeBytes(b)))
	}
	return b
}

func (d decValueEncoder) Decode(b []byte) sdk.Dec {
	dec := new(sdk.Dec)
	err := dec.Unmarshal(b)
	if err != nil {
		panic(fmt.Errorf("%w %s", err, HumanizeBytes(b)))
	}
	return *dec
}

func (d decValueEncoder) Stringify(value sdk.Dec) string {
	return value.String()
}

func (d decValueEncoder) Name() string {
	return "sdk.Dec"
}

// AccAddressValueEncoder ValueEncoder[sdk.AccAddress]

type accAddressValueEncoder struct{}

func (a accAddressValueEncoder) Encode(value sdk.AccAddress) []byte    { return value }
func (a accAddressValueEncoder) Decode(b []byte) sdk.AccAddress        { return b }
func (a accAddressValueEncoder) Stringify(value sdk.AccAddress) string { return value.String() }
func (a accAddressValueEncoder) Name() string                          { return "sdk.AccAddress" }

// IntValueEncoder ValueEncoder[sdk.Int]

type intValueEncoder struct{}

func (intValueEncoder) Encode(value sdkmath.Int) []byte {
	bz, err := value.Marshal()
	if err != nil {
		panic(fmt.Errorf("invalid math.Int %s: %w", value, err))
	}
	return bz
}

func (intValueEncoder) Decode(b []byte) sdkmath.Int {
	n := new(sdkmath.Int)
	if err := n.Unmarshal(b); err != nil {
		panic(fmt.Errorf("decoding math.Int from bytes failed: %w", err))
	}
	return *n
}

func (intValueEncoder) Stringify(value sdkmath.Int) string {
	return value.String()
}

func (intValueEncoder) Name() string {
	return "math.Int (signed)"
}

// UintValueEncoder ValueEncoder[sdk.Uint]

type uintValueEncoder struct{}

func (uintValueEncoder) Encode(value sdkmath.Uint) []byte {
	if value.IsNil() {
		panic("cannot encode invalid math.Uint")
	}
	return encodeFixedWidthUnsignedBigEndian(value.BigInt())
}

func (uintValueEncoder) Decode(b []byte) sdkmath.Uint {
	return sdkmath.NewUintFromBigInt(decodeFixedWidthUnsignedBigEndian(b))
}

func (uintValueEncoder) Stringify(value sdkmath.Uint) string {
	return value.String()
}

func (uintValueEncoder) Name() string {
	return "math.Uint (fixed-width BE)"
}

// IntKeyEncoder KeyEncoder[sdk.Int]

var IntKeyEncoder KeyEncoder[sdkmath.Int] = intKeyEncoder{}

type intKeyEncoder struct{}

const maxIntKeyLen = sdkmath.MaxBitLen / 8

func (intKeyEncoder) Encode(key sdkmath.Int) []byte {
	if key.IsNil() {
		panic("cannot encode invalid math.Int")
	}
	if key.IsNegative() {
		panic("cannot encode negative math.Int")
	}
	return encodeFixedWidthUnsignedBigEndian(key.BigInt())
}

func (intKeyEncoder) Decode(b []byte) (int, sdkmath.Int) {
	return maxIntKeyLen, sdkmath.NewIntFromBigInt(decodeFixedWidthUnsignedBigEndian(b))
}

func (intKeyEncoder) Stringify(key sdkmath.Int) string { return key.String() }

func encodeFixedWidthUnsignedBigEndian(i *big.Int) []byte {
	be := i.Bytes()
	padded := make([]byte, maxIntKeyLen)
	copy(padded[maxIntKeyLen-len(be):], be)
	return padded
}

func decodeFixedWidthUnsignedBigEndian(b []byte) *big.Int {
	if len(b) != maxIntKeyLen {
		panic("invalid fixed-width uint bytes length")
	}
	return new(big.Int).SetBytes(b)
}
