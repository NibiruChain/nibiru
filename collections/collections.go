package collections

import "github.com/cosmos/cosmos-sdk/codec"

type Object interface {
	codec.ProtoMarshaler
}

type noOpObject struct{}

func (n noOpObject) Reset() {
	panic("must never be called")
}

func (n noOpObject) String() string {
	panic("must never be called")
}

func (n noOpObject) ProtoMessage() {
	panic("must never be called")
}

func (n noOpObject) Marshal() ([]byte, error) {
	panic("must never be called")
}

func (n noOpObject) MarshalTo(_ []byte) (_ int, _ error) {
	panic("must never be called")
}

func (n noOpObject) MarshalToSizedBuffer(_ []byte) (int, error) {
	panic("must never be called")
}

func (n noOpObject) Size() int {
	panic("must never be called")
}

func (n noOpObject) Unmarshal(data []byte) error {
	panic("must never be called")
}

var _ Object = (*noOpObject)(nil)
