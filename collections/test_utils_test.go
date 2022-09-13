package collections

import (
	"encoding/json"
	"github.com/NibiruChain/nibiru/collections/keys"
)

var _ Object = (*lock)(nil)

type lock struct {
	Start   keys.Uint8Key
	End     keys.Uint8Key
	ID      keys.Uint8Key
	Address keys.StringKey
}

func (l lock) Reset() {
	//TODO implement me
	panic("implement me")
}

func (l lock) String() string {
	//TODO implement me
	panic("implement me")
}

func (l lock) ProtoMessage() {
	//TODO implement me
	panic("implement me")
}

func (l *lock) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

func (l *lock) MarshalTo(data []byte) (n int, err error) {
	panic("implement me")
}

func (l *lock) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (l lock) Size() int {
	panic("implement me")
}

func (l lock) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &l)
}
