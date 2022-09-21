package collections

import (
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// Object defines an object which can marshal and unmarshal itself to and from bytes.
type Object interface {
	// Marshal marshals the object into bytes.
	Marshal() (b []byte, err error)
	// Unmarshal populates the object from bytes.
	Unmarshal(b []byte) error
}

// storeCodec implements only the subset of functionalities
// required for the ser/de at state layer.
// It respects cosmos-sdk guarantees around interface unpacking.
type storeCodec struct {
	ir codectypes.InterfaceRegistry
}

func newStoreCodec(cdc codec.BinaryCodec) storeCodec {
	return storeCodec{ir: cdc.(*codec.ProtoCodec).InterfaceRegistry()}
}

func (c storeCodec) marshal(o Object) []byte {
	bytes, err := o.Marshal()
	if err != nil {
		panic(err)
	}
	return bytes
}

func (c storeCodec) unmarshal(bytes []byte, o Object) {
	err := o.Unmarshal(bytes)
	if err != nil {
		panic(err)
	}
	err = codectypes.UnpackInterfaces(o, c.ir)
	if err != nil {
		panic(err)
	}
}

// TODO(mercilex): improve typeName api
func typeName(o Object) string {
	switch o.(type) {
	case *nilObject, nilObject:
		return "no-op-object"
	}
	pm, ok := o.(proto.Message)
	if !ok {
		return "unknown"
	}
	return proto.MessageName(pm)
}
