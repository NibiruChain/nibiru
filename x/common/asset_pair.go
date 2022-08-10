package common

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strings"
)

const (
	PairSeparator = ":"
)

var (
	_ sdk.CustomProtobufType = (*AssetPair)(nil)
)

func NewAssetPairFromTokens(t0, t1 string) (AssetPair, error) {
	err := sdk.ValidateDenom(t0)
	if err != nil {
		return AssetPair{}, err
	}
	err = sdk.ValidateDenom(t1)
	if err != nil {
		return AssetPair{}, err
	}

	return AssetPair{
		t0:  t0,
		t1:  t1,
		raw: fmt.Sprintf("%s%s%s", t0, PairSeparator, t1),
	}, nil
}

// MustNewAssetPairFromTokens instantiates a new AssetPair from the provided
// token pair. It panics if the provided denom names are not valid.
func MustNewAssetPairFromTokens(t0, t1 string) AssetPair {
	ap, err := NewAssetPairFromTokens(t0, t1)
	if err != nil {
		panic(err)
	}

	return ap
}

// NewAssetPair returns a new asset pair instance if the pair is valid.
// The form, "token0:token1", is expected for 'pair'.
// Use this function to return an error instead of panicking.
func NewAssetPair(pair string) (AssetPair, error) {
	split := strings.Split(pair, PairSeparator)
	splitLen := len(split)
	if splitLen != 2 {
		if splitLen == 1 {
			return AssetPair{}, sdkerrors.Wrapf(ErrInvalidTokenPair,
				"pair separator missing for pair name, %v", pair)
		} else {
			return AssetPair{}, sdkerrors.Wrapf(ErrInvalidTokenPair,
				"pair name %v must have exactly two assets, not %v", pair, splitLen)
		}
	}

	if split[0] == "" || split[1] == "" {
		return AssetPair{}, sdkerrors.Wrapf(ErrInvalidTokenPair,
			"empty token identifiers are not allowed. token0: %v, token1: %v.",
			split[0], split[1])
	}

	// validate token0
	err := sdk.ValidateDenom(split[0])
	if err != nil {
		return AssetPair{}, err
	}

	// validate token1
	err = sdk.ValidateDenom(split[1])
	if err != nil {
		return AssetPair{}, err
	}

	return AssetPair{t0: split[0], t1: split[1], raw: pair}, nil
}

// MustNewAssetPair returns a new asset pair. It will panic if 'pair' is invalid.
// The form, "token0:token1", is expected for 'pair'.
func MustNewAssetPair(pair string) AssetPair {
	assetPair, err := NewAssetPair(pair)
	if err != nil {
		panic(err)
	}
	return assetPair
}

type AssetPair struct {
	t0, t1, raw string
}

func (a AssetPair) String() string {
	return a.raw
}

// SortedName is the string representation of the pair with sorted assets.
func (a AssetPair) SortedName() string {
	return SortedPairNameFromDenoms([]string{a.t0, a.t1})
}

func (a AssetPair) IsSortedOrder() bool {
	return a.SortedName() == a.String()
}

func (a AssetPair) Inverse() AssetPair {
	inverse, err := NewAssetPairFromTokens(a.t1, a.t0)
	// this must never happen as there cannot
	// be invalid initialization of AssetPair
	if err != nil {
		panic(err)
	}
	return inverse
}

func (a AssetPair) Token0() string {
	return a.t0
}

func (a AssetPair) BaseDenom() string {
	return a.t0
}

func (a AssetPair) Token1() string {
	return a.t1
}

func (a AssetPair) QuoteDenom() string {
	return a.t1
}

// custom type implementation

func (a AssetPair) Marshal() ([]byte, error) {
	return []byte(a.raw), nil
}

func (a AssetPair) MarshalTo(data []byte) (n int, err error) {
	b, err := a.Marshal()
	if err != nil {
		return 0, err
	}

	copy(data, b)
	return len(b), nil
}

func (a *AssetPair) Unmarshal(data []byte) error {
	b, err := NewAssetPair(string(data))
	if err != nil {
		return err
	}
	*a = b
	return nil
}

func (a AssetPair) Size() int {
	return len(a.raw)
}

func (a AssetPair) MarshalJSON() ([]byte, error) {
	return []byte("\"" + a.raw + "\""), nil
}

func (a *AssetPair) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("invalid json string")
	}
	if data[0] != '"' && data[len(data)-1] != '"' {
		return fmt.Errorf("json string expected")
	}

	x, err := NewAssetPair(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*a = x
	return nil
}

func (a AssetPair) Equal(i interface{}) bool {
	switch m := i.(type) {
	case AssetPair:
		return m.t0 == a.t0 && m.t1 == a.t1 && m.raw == a.raw
	case *AssetPair:
		return m.t0 == a.t0 && m.t1 == a.t1 && m.raw == a.raw
	default:
		return false
	}
}
