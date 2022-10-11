package collections

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var DecValueEncoder = decValueEncoder{}

type decValueEncoder struct{}

func (d decValueEncoder) ValueEncode(value sdk.Dec) []byte {
	b, err := value.Marshal()
	if err != nil {
		panic(err)
	}
	return b
}

func (d decValueEncoder) ValueDecode(b []byte) sdk.Dec {
	dec := new(sdk.Dec)
	err := dec.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return *dec
}

func (d decValueEncoder) Stringify(value sdk.Dec) string {
	return value.String()
}

func (d decValueEncoder) Name() string {
	return "sdk.Dec"
}
