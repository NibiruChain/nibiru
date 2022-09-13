package collections

import (
	"bytes"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/gogo/protobuf/proto"
)

type Object interface {
	codec.ProtoMarshaler
}

// setObject is used when no object functionality is needed.
type setObject struct{}

func (n setObject) Reset() {
	panic("must never be called")
}

func (n setObject) String() string {
	panic("must never be called")
}

func (n setObject) ProtoMessage() {
	panic("must never be called")
}

func (n setObject) Marshal() ([]byte, error) {
	return []byte{}, nil
}

func (n setObject) MarshalTo(_ []byte) (_ int, _ error) {
	panic("must never be called")
}

func (n setObject) MarshalToSizedBuffer(_ []byte) (int, error) {
	panic("must never be called")
}

func (n setObject) Size() int {
	panic("must never be called")
}

func (n setObject) Unmarshal(b []byte) error {
	if !bytes.Equal(b, []byte{}) {
		panic("bad usage")
	}
	return nil
}

var _ Object = (*setObject)(nil)

func typeName(o Object) string {
	switch o.(type) {
	case *setObject, setObject:
		return "no-op-object"
	}
	n := proto.MessageName(o)
	if n == "" {
		panic("invalid Object implementation")
	}
	return n
}
