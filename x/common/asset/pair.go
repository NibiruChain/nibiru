package asset

import (
	"encoding/json"
	"fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// paired against USD
var ErrInvalidTokenPair = sdkerrors.Register("asset", 1, "invalid token pair")

type Pair string

func NewPair(base string, quote string) Pair {
	// validate as denom
	ap := fmt.Sprintf("%s%s%s", base, ":", quote)
	return Pair(ap)
}

// TryNewPair New returns a new asset pair instance if the pair is valid.
// The form, "token0:token1", is expected for 'pair'.
// Use this function to return an error instead of panicking.
func TryNewPair(pair string) (Pair, error) {
	split := strings.Split(pair, ":")
	splitLen := len(split)
	if splitLen != 2 {
		if splitLen == 1 {
			return "", sdkerrors.Wrapf(ErrInvalidTokenPair,
				"pair separator missing for pair name, %v", pair)
		} else {
			return "", sdkerrors.Wrapf(ErrInvalidTokenPair,
				"pair name %v must have exactly two assets, not %v", pair, splitLen)
		}
	}

	if split[0] == "" || split[1] == "" {
		return "", sdkerrors.Wrapf(ErrInvalidTokenPair,
			"empty token identifiers are not allowed. token0: %v, token1: %v.",
			split[0], split[1])
	}

	// validate as denom
	Pair := NewPair(split[0], split[1])
	return Pair, Pair.Validate()
}

// MustNewPair returns a new asset pair. It will panic if 'pair' is invalid.
// The form, "token0:token1", is expected for 'pair'.
func MustNewPair(pair string) Pair {
	Pair, err := TryNewPair(pair)
	if err != nil {
		panic(err)
	}
	return Pair
}

/*
String returns the string representation of the asset pair.

Note that this differs from the output of the proto-generated 'String' method.
*/
func (pair Pair) String() string {
	return string(pair)
}

func (pair Pair) Inverse() Pair {
	return NewPair(pair.QuoteDenom(), pair.BaseDenom())
}

func (pair Pair) BaseDenom() string {
	split := strings.Split(pair.String(), ":")
	return split[0]
}

func (pair Pair) QuoteDenom() string {
	split := strings.Split(pair.String(), ":")
	return split[1]
}

// Validate performs a basic validation of the market params
func (pair Pair) Validate() error {
	split := strings.Split(pair.String(), ":")
	if len(split) != 2 {
		return ErrInvalidTokenPair.Wrap(pair.String())
	}

	if err := sdk.ValidateDenom(split[0]); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid base asset: %s", err)
	}
	if err := sdk.ValidateDenom(split[1]); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid quote asset: %s", err)
	}
	return nil
}

func (pair Pair) Equal(other Pair) bool {
	return pair.String() == other.String()
}

var _ sdk.CustomProtobufType = (*Pair)(nil)

func (pair Pair) Marshal() ([]byte, error) {
	return []byte(pair), nil
}

func (pair *Pair) Unmarshal(data []byte) error {
	*pair = Pair(data)
	return nil
}

func (pair Pair) MarshalJSON() ([]byte, error) {
	return json.Marshal(pair.String())
}

func (pair *Pair) UnmarshalJSON(data []byte) error {
	var pairString string
	if err := json.Unmarshal(data, &pairString); err != nil {
		return err
	}
	*pair = Pair(pairString)
	return nil
}

func (pair Pair) MarshalTo(data []byte) (n int, err error) {
	copy(data, pair)
	return pair.Size(), nil
}

func (pair Pair) Size() int {
	return len(pair)
}

var PairKeyEncoder collections.KeyEncoder[Pair] = pairKeyEncoder{}

type pairKeyEncoder struct{}

func (pairKeyEncoder) Stringify(a Pair) string { return a.String() }
func (pairKeyEncoder) Encode(a Pair) []byte {
	return collections.StringKeyEncoder.Encode(a.String())
}

func (pairKeyEncoder) Decode(b []byte) (int, Pair) {
	i, s := collections.StringKeyEncoder.Decode(b)
	return i, MustNewPair(s)
}

// MustNewPairs constructs a new asset pair set. A panic will occur if one of
// the provided pair names is invalid.
func MustNewPairs(pairStrings ...string) (pairs []Pair) {
	for _, pairString := range pairStrings {
		pairs = append(pairs, MustNewPair(pairString))
	}
	return pairs
}

func PairsToStrings(pairs []Pair) []string {
	pairsStrings := []string{}
	for _, pair := range pairs {
		pairsStrings = append(pairsStrings, pair.String())
	}
	return pairsStrings
}
