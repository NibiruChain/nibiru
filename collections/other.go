package collections

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var Uint64ValueEncoder = uint64Value{}

var DecValueEncoder = decValueEncoder{}

type valAddressKeyEncoder struct{}

func (v valAddressKeyEncoder) KeyEncode(key sdk.ValAddress) []byte {
	return Keys.String.KeyEncode(key.String())
}
func (v valAddressKeyEncoder) KeyDecode(b []byte) (int, sdk.ValAddress) {
	r, s := Keys.String.KeyDecode(b)
	valAddr, err := sdk.ValAddressFromBech32(s)
	if err != nil {
		panic(err)
	}
	return r, valAddr
}
func (v valAddressKeyEncoder) Stringify(key sdk.ValAddress) string { return key.String() }

var ValAddressKeyEncoder KeyEncoder[sdk.ValAddress] = valAddressKeyEncoder{}

type accAddressValueEncoder struct{}

func (a accAddressValueEncoder) ValueEncode(value sdk.AccAddress) []byte { return value }
func (a accAddressValueEncoder) ValueDecode(b []byte) sdk.AccAddress     { return b }
func (a accAddressValueEncoder) Stringify(value sdk.AccAddress) string   { return value.String() }
func (a accAddressValueEncoder) Name() string                            { return "sdk.AccAddress" }

var AccAddressValueEncoder ValueEncoder[sdk.AccAddress] = accAddressValueEncoder{}

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
