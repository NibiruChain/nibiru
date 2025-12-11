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
	return IntKeyEncoder.Encode(value)
}

func (intValueEncoder) Decode(b []byte) sdkmath.Int {
	_, got := IntKeyEncoder.Decode(b)
	return got
}

func (intValueEncoder) Stringify(value sdkmath.Int) string {
	return IntKeyEncoder.Stringify(value)
}

func (intValueEncoder) Name() string {
	return "math.Int"
}

// IntKeyEncoder

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
	i := key.BigInt()

	be := i.Bytes()
	padded := make([]byte, maxIntKeyLen)
	copy(padded[maxIntKeyLen-len(be):], be)
	return padded
}

func (intKeyEncoder) Decode(b []byte) (int, sdkmath.Int) {
	if len(b) != maxIntKeyLen {
		panic("invalid key length")
	}
	i := new(big.Int).SetBytes(b)
	return maxIntKeyLen, sdkmath.NewIntFromBigInt(i)
}

func (intKeyEncoder) Stringify(key sdkmath.Int) string { return key.String() }
