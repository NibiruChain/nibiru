package common

import (
	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// LegacyDecValue represents a collections.ValueCodec to work with LegacyDec.
	LegacyDecValue  collcodec.ValueCodec[sdkmath.LegacyDec] = legacyDecValueCodec{}
	SdkIntKey       collcodec.KeyCodec[sdkmath.Int]         = mathIntKeyCodec{}
	AccAddressValue collcodec.ValueCodec[sdk.AccAddress]    = accAddressValueCodec{}
)

// Collection Codecs

// math.LegacyDec Value Codec

type legacyDecValueCodec struct{}

func (cdc legacyDecValueCodec) Encode(value sdkmath.LegacyDec) ([]byte, error) {
	return value.Marshal()
}

func (cdc legacyDecValueCodec) Decode(buffer []byte) (sdkmath.LegacyDec, error) {
	v := sdkmath.LegacyZeroDec()
	err := v.Unmarshal(buffer)
	return v, err
}

func (cdc legacyDecValueCodec) EncodeJSON(value sdkmath.LegacyDec) ([]byte, error) {
	return value.MarshalJSON()
}

func (cdc legacyDecValueCodec) DecodeJSON(buffer []byte) (sdkmath.LegacyDec, error) {
	v := sdkmath.LegacyDec{}
	err := v.UnmarshalJSON(buffer)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	return v, nil
}

func (cdc legacyDecValueCodec) Stringify(value sdkmath.LegacyDec) string {
	return value.String()
}

func (cdc legacyDecValueCodec) ValueType() string {
	return "math.LegacyDec"
}

// AccAddress Value Codec

type accAddressValueCodec struct{}

func (cdc accAddressValueCodec) Encode(value sdk.AccAddress) ([]byte, error) {
	return value.Marshal()
}

func (cdc accAddressValueCodec) Decode(buffer []byte) (sdk.AccAddress, error) {
	v := sdk.AccAddress{}
	err := v.Unmarshal(buffer)
	return v, err
}

func (cdc accAddressValueCodec) EncodeJSON(value sdk.AccAddress) ([]byte, error) {
	return value.MarshalJSON()
}

func (cdc accAddressValueCodec) DecodeJSON(buffer []byte) (sdk.AccAddress, error) {
	v := sdk.AccAddress{}
	err := v.UnmarshalJSON(buffer)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (cdc accAddressValueCodec) Stringify(value sdk.AccAddress) string {
	return value.String()
}

func (cdc accAddressValueCodec) ValueType() string {
	return "sdk.AccAddress"
}

// math.Int Key Codec

type mathIntKeyCodec struct {
	stringDecoder func(string) (sdkmath.Int, error)
	keyType       string
}

func (cdc mathIntKeyCodec) Encode(buffer []byte, key sdkmath.Int) (int, error) {
	bytes, err := key.Marshal()
	if err != nil {
		return 0, err
	}
	copy(bytes, buffer)
	return key.Size(), nil
}

func (cdc mathIntKeyCodec) Decode(buffer []byte) (int, sdkmath.Int, error) {
	v := sdkmath.ZeroInt()
	err := v.Unmarshal(buffer)
	if err != nil {
		return 0, v, err
	}
	return v.Size(), v, nil
}

func (cdc mathIntKeyCodec) Size(key sdkmath.Int) int {
	return key.Size()
}

func (cdc mathIntKeyCodec) EncodeJSON(value sdkmath.Int) ([]byte, error) {
	return collections.StringKey.EncodeJSON(value.String())
}

func (cdc mathIntKeyCodec) DecodeJSON(b []byte) (v sdkmath.Int, err error) {
	s, err := collections.StringKey.DecodeJSON(b)
	if err != nil {
		return
	}
	v, err = cdc.stringDecoder(s)
	return
}

func (cdc mathIntKeyCodec) Stringify(key sdkmath.Int) string {
	return key.String()
}

func (cdc mathIntKeyCodec) KeyType() string {
	return cdc.keyType
}

func (cdc mathIntKeyCodec) EncodeNonTerminal(buffer []byte, key sdkmath.Int) (int, error) {
	return cdc.Encode(buffer, key)
}

func (cdc mathIntKeyCodec) DecodeNonTerminal(buffer []byte) (int, sdkmath.Int, error) {
	return cdc.Decode(buffer)
}

func (cdc mathIntKeyCodec) SizeNonTerminal(key sdkmath.Int) int {
	return key.Size()
}
