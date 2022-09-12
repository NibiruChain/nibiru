package common

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	DenomGov    = "unibi"
	DenomColl   = "uusdc"
	DenomStable = "unusd"
	DenomBTC    = "ubtc"
	DenomETH    = "ueth"

	ModuleName = "common"

	TreasuryPoolModuleAccount = "treasury_pool"

	PairSeparator = ":"
)

var (
	PairGovStable  = AssetPair{Token0: DenomGov, Token1: DenomStable}
	PairCollStable = AssetPair{Token0: DenomColl, Token1: DenomStable}
	PairBTCStable  = AssetPair{Token0: DenomBTC, Token1: DenomStable}
	PairETHStable  = AssetPair{Token0: DenomETH, Token1: DenomStable}

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

// SortedName is the string representation of the pair with sorted assets.
func (pair AssetPair) SortedName() string {
	return SortedPairNameFromDenoms([]string{pair.Token0, pair.Token1})
}

/*
	String returns the string representation of the asset pair.

Note that this differs from the output of the proto-generated 'String' method.
*/
func (pair AssetPair) String() string {
	return fmt.Sprintf("%s%s%s", pair.Token0, PairSeparator, pair.Token1)
}

func (pair AssetPair) IsSortedOrder() bool {
	return pair.SortedName() == pair.String()
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

func DenomsFromPoolName(pool string) (denoms []string) {
	return strings.Split(pool, ":")
}

// SortedPairNameFromDenoms returns a sorted string representing a pool of assets
func SortedPairNameFromDenoms(denoms []string) string {
	sort.Strings(denoms) // alphabetically sort in-place
	return PairNameFromDenoms(denoms)
}

// PairNameFromDenoms returns a string representing a pool of assets in the
// exact order the denoms were given as args
func PairNameFromDenoms(denoms []string) string {
	poolName := denoms[0]
	for idx, denom := range denoms {
		if idx != 0 {
			poolName += fmt.Sprintf("%s%s", PairSeparator, denom)
		}
	}
	return poolName
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
func (pairs AssetPairs) Contains(pair AssetPair) bool {
	isContained, _ := pairs.ContainsAtIndex(pair)
	return isContained
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

// ContainsAtIndex checks if a token pair is contained within 'Pairs' and
// a boolean for this condition alongside the corresponding index of 'pair' in
// the slice of pairs.
func (pairs AssetPairs) ContainsAtIndex(pair AssetPair) (bool, int) {
	for idx, element := range pairs {
		if (element.Token0 == pair.Token0) && (element.Token1 == pair.Token1) {
			return true, idx
		}
	}
	return false, -1
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
