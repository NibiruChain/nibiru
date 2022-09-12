package collections

import "github.com/cosmos/cosmos-sdk/codec"

type Object interface {
	codec.ProtoMarshaler
}

// panicObject is used when no object functionality must ever be called.
type panicObject struct{}

func (n panicObject) Reset() {
	panic("must never be called")
}

func (n panicObject) String() string {
	panic("must never be called")
}

func (n panicObject) ProtoMessage() {
	panic("must never be called")
}

func (n panicObject) Marshal() ([]byte, error) {
	panic("must never be called")
}

func (n panicObject) MarshalTo(_ []byte) (_ int, _ error) {
	panic("must never be called")
}

func (n panicObject) MarshalToSizedBuffer(_ []byte) (int, error) {
	panic("must never be called")
}

func (n panicObject) Size() int {
	panic("must never be called")
}

func (n panicObject) Unmarshal(data []byte) error {
	panic("must never be called")
}

var _ Object = (*panicObject)(nil)
