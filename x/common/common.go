package common

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
)

var (
	PairGovStable  = MustNewAssetPairFromTokens(DenomGov, DenomStable)
	PairCollStable = MustNewAssetPairFromTokens(DenomColl, DenomStable)
	PairBTCStable  = MustNewAssetPairFromTokens(DenomBTC, DenomStable)
	PairETHStable  = MustNewAssetPairFromTokens(DenomETH, DenomStable)

	ErrInvalidTokenPair = sdkerrors.Register(ModuleName, 1, "invalid token pair")
)

//-----------------------------------------------------------------------------
// AssetPair

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
		seenPairs[pairID] = true
	}
	return nil
}

// ContainsAtIndex checks if a token pair is contained within 'Pairs' and
// a boolean for this condition alongside the corresponding index of 'pair' in
// the slice of pairs.
func (pairs AssetPairs) ContainsAtIndex(pair AssetPair) (bool, int) {
	for idx, element := range pairs {
		if (element.Token0() == pair.Token0()) && (element.Token1() == pair.Token1()) {
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
