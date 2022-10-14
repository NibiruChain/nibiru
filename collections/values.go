package collections

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

var (
	AccAddressValueEncoder ValueEncoder[sdk.AccAddress] = accAddressValueEncoder{}
	DecValueEncoder        ValueEncoder[sdk.Dec]        = decValue{}
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

type decValue struct{}

func (d decValue) Encode(value sdk.Dec) []byte {
	b, err := value.Marshal()
	if err != nil {
		panic(err)
	}
	return b
}

func (d decValue) Decode(b []byte) sdk.Dec {
	dec := new(sdk.Dec)
	err := dec.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return *dec
}

func (d decValue) Stringify(value sdk.Dec) string {
	return value.String()
}

func (d decValue) Name() string {
	return "sdk.Dec"
}

type accAddressValueEncoder struct{}

func (a accAddressValueEncoder) Encode(value sdk.AccAddress) []byte    { return value }
func (a accAddressValueEncoder) Decode(b []byte) sdk.AccAddress        { return b }
func (a accAddressValueEncoder) Stringify(value sdk.AccAddress) string { return value.String() }
func (a accAddressValueEncoder) Name() string                          { return "sdk.AccAddress" }
