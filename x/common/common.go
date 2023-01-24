package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/NibiruChain/collections"
	"github.com/holiman/uint256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type AssetPair string

const (
	// stablecoins
	DenomUSDC = "uusdc"
	DenomNUSD = "unusd"
	DenomUSD  = "uusd"
	DenomUSDT = "uusdt"

	// crypto assets
	DenomNIBI = "unibi"
	DenomBTC  = "ubtc"
	DenomETH  = "ueth"

	ModuleName = "common"

	TreasuryPoolModuleAccount = "treasury_pool"

	PairSeparator = ":"
)

var (
	// paired against USD
	Pair_NIBI_USD = NewAssetPair(DenomNIBI, DenomUSD)
	Pair_USDC_USD = NewAssetPair(DenomUSDC, DenomUSD)
	Pair_BTC_USD  = NewAssetPair(DenomBTC, DenomUSD)
	Pair_ETH_USD  = NewAssetPair(DenomETH, DenomUSD)

	// paired against NUSD
	Pair_NIBI_NUSD = NewAssetPair(DenomNIBI, DenomNUSD)
	Pair_USDC_NUSD = NewAssetPair(DenomUSDC, DenomNUSD)
	Pair_BTC_NUSD  = NewAssetPair(DenomBTC, DenomNUSD)
	Pair_ETH_NUSD  = NewAssetPair(DenomETH, DenomNUSD)

	ErrInvalidTokenPair = sdkerrors.Register(ModuleName, 1, "invalid token pair")
	APrecision          = uint256.NewInt().SetUint64(1)

	// Precision for int representation in sdk.Int objects
	Precision = int64(1_000_000)
)

//-----------------------------------------------------------------------------
// AssetPair

func NewAssetPair(base string, quote string) AssetPair {
	// validate as denom
	ap := fmt.Sprintf("%s%s%s", base, PairSeparator, quote)
	return AssetPair(ap)
}

// NewAssetPair returns a new asset pair instance if the pair is valid.
// The form, "token0:token1", is expected for 'pair'.
// Use this function to return an error instead of panicking.
func TryNewAssetPair(pair string) (AssetPair, error) {
	split := strings.Split(pair, PairSeparator)
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
	assetPair := NewAssetPair(split[0], split[1])
	return assetPair, assetPair.Validate()
}

// MustNewAssetPair returns a new asset pair. It will panic if 'pair' is invalid.
// The form, "token0:token1", is expected for 'pair'.
func MustNewAssetPair(pair string) AssetPair {
	assetPair, err := TryNewAssetPair(pair)
	if err != nil {
		panic(err)
	}
	return assetPair
}

/*
String returns the string representation of the asset pair.

Note that this differs from the output of the proto-generated 'String' method.
*/
func (pair AssetPair) String() string {
	return string(pair)
}

func (pair AssetPair) Inverse() AssetPair {
	return NewAssetPair(pair.QuoteDenom(), pair.BaseDenom())
}

func (pair AssetPair) BaseDenom() string {
	split := strings.Split(pair.String(), PairSeparator)
	return split[0]
}

func (pair AssetPair) QuoteDenom() string {
	split := strings.Split(pair.String(), PairSeparator)
	return split[1]
}

// Validate performs a basic validation of the market params
func (pair AssetPair) Validate() error {
	if err := sdk.ValidateDenom(pair.BaseDenom()); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid base asset: %s", err)
	}
	if err := sdk.ValidateDenom(pair.QuoteDenom()); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid quote asset: %s", err)
	}
	return nil
}

func (pair AssetPair) Equal(other AssetPair) bool {
	return pair.String() == other.String()
}

var _ sdk.CustomProtobufType = (*AssetPair)(nil)

func (pair AssetPair) Marshal() ([]byte, error) {
	return []byte(pair.String()), nil
}

func (pair *AssetPair) Unmarshal(data []byte) error {
	*pair = AssetPair(data)
	return nil
}

func (pair AssetPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(pair.String())
}

func (pair *AssetPair) UnmarshalJSON(data []byte) error {
	var pairString string
	if err := json.Unmarshal(data, &pairString); err != nil {
		return err
	}
	*pair = AssetPair(pairString)
	return nil
}

func (pair AssetPair) MarshalTo(data []byte) (n int, err error) {
	copy(data, pair.String())
	return pair.Size(), nil
}

func (pair AssetPair) Size() int {
	return len(pair.String())
}

var AssetPairKeyEncoder collections.KeyEncoder[AssetPair] = assetPairKeyEncoder{}

type assetPairKeyEncoder struct{}

func (assetPairKeyEncoder) Stringify(a AssetPair) string { return a.String() }
func (assetPairKeyEncoder) Encode(a AssetPair) []byte {
	return collections.StringKeyEncoder.Encode(a.String())
}
func (assetPairKeyEncoder) Decode(b []byte) (int, AssetPair) {
	i, s := collections.StringKeyEncoder.Decode(b)
	return i, MustNewAssetPair(s)
}

//-----------------------------------------------------------------------------
// AssetPairs

// AssetPairs is a set of AssetPair, one per pair.
type AssetPairs []AssetPair

// NewAssetPairs constructs a new asset pair set. A panic will occur if one of
// the provided pair names is invalid.
func NewAssetPairs(pairStrings ...string) (pairs AssetPairs) {
	for _, pairString := range pairStrings {
		pairs = append(pairs, MustNewAssetPair(pairString))
	}
	return pairs
}

// Contains checks if a token pair is contained within 'Pairs'
func (haystack AssetPairs) Contains(needle AssetPair) bool {
	for _, p := range haystack {
		if p.Equal(needle) {
			return true
		}
	}
	return false
}

func (pairs AssetPairs) Strings() []string {
	pairsStrings := []string{}
	for _, pair := range pairs {
		pairsStrings = append(pairsStrings, pair.String())
	}
	return pairsStrings
}

func (pairs AssetPairs) Validate() error {
	seenPairs := make(map[string]bool)
	for _, pair := range pairs {
		pairID := pair.String()
		if seenPairs[pairID] {
			return fmt.Errorf("duplicate pair %s", pairID)
		}
		if err := pair.Validate(); err != nil {
			return err
		}
		seenPairs[pairID] = true
	}
	return nil
}

type assetPairsJSON AssetPairs

// MarshalJSON implements a custom JSON marshaller for the AssetPairs type to allow
// nil AssetPairs to be encoded as empty
func (pairs AssetPairs) MarshalJSON() ([]byte, error) {
	if pairs == nil {
		return json.Marshal(assetPairsJSON(AssetPairs{}))
	}
	return json.Marshal(assetPairsJSON(pairs))
}
