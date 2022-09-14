package common

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	DenomNIBI = "unibi"
	DenomUSDC = "uusdc"
	DenomNUSD = "unusd"
	DenomBTC  = "ubtc"
	DenomETH  = "ueth"

	ModuleName = "common"

	TreasuryPoolModuleAccount = "treasury_pool"

	PairSeparator = ":"
)

var (
	Pair_NIBI_NUSD = AssetPair{Token0: DenomNIBI, Token1: DenomNUSD}
	Pair_USDC_NUSD = AssetPair{Token0: DenomUSDC, Token1: DenomNUSD}
	Pair_BTC_NUSD  = AssetPair{Token0: DenomBTC, Token1: DenomNUSD}
	Pair_ETH_NUSD  = AssetPair{Token0: DenomETH, Token1: DenomNUSD}

	ErrInvalidTokenPair = sdkerrors.Register(ModuleName, 1, "invalid token pair")
)

//-----------------------------------------------------------------------------
// AssetPair

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

	return AssetPair{Token0: split[0], Token1: split[1]}, nil
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

/*
String returns the string representation of the asset pair.

Note that this differs from the output of the proto-generated 'String' method.
*/
func (pair AssetPair) String() string {
	return fmt.Sprintf("%s%s%s", pair.Token0, PairSeparator, pair.Token1)
}

func (pair AssetPair) Inverse() AssetPair {
	return AssetPair{pair.Token1, pair.Token0}
}

func (pair AssetPair) BaseDenom() string {
	return pair.Token0
}

func (pair AssetPair) QuoteDenom() string {
	return pair.Token1
}

// Validate performs a basic validation of the market params
func (pair AssetPair) Validate() error {
	if err := sdk.ValidateDenom(pair.Token1); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid token1 asset: %s", err)
	}
	if err := sdk.ValidateDenom(pair.Token0); err != nil {
		return ErrInvalidTokenPair.Wrapf("invalid token0 asset: %s", err)
	}
	return nil
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

func ToSdkPointer(num interface{}) interface{} {
	switch sdkType := num.(type) {
	case sdk.Int:
		pointer := new(sdk.Int)
		*pointer = num.(sdk.Int)
		return pointer
	case sdk.Dec:
		pointer := new(sdk.Dec)
		*pointer = num.(sdk.Dec)
		return pointer
	default:
		errMsg := fmt.Errorf("type passed must be sdk.Int or sdk.Dec, not %s", sdkType)
		panic(errMsg)
	}
}
