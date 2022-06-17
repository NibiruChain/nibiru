package common

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	GovDenom        = "unibi"
	CollDenom       = "uust"
	StableDenom     = "unusd"
	StakeTokenDenom = "stake"
	TestTokenDenom  = "test"

	TreasuryPoolModuleAccount = "treasury_pool"

	PairSeparator = ":"

	WhitelistedColl = []string{CollDenom}

	GovStablePool  = AssetPair{Token0: GovDenom, Token1: StableDenom}
	CollStablePool = AssetPair{Token0: CollDenom, Token1: StableDenom}
	TestStablePool = AssetPair{Token0: TestTokenDenom, Token1: StableDenom}

	ErrInvalidTokenPair = fmt.Errorf("invalid token pair")
)

var (
	PAIR_uBTC_uNUSD = AssetPair{Token0: "uBTC", Token1: "uNUSD"}
	PAIR_uETH_uNUSD = AssetPair{Token0: "uETH", Token1: "uNUSD"}
)

func NewAssetPairFromStr(pair string) (AssetPair, error) {
	split := strings.Split(pair, PairSeparator)
	if len(split) != 2 {
		return AssetPair{}, ErrInvalidTokenPair
	}

	if split[0] == "" || split[1] == "" {
		return AssetPair{}, ErrInvalidTokenPair
	}

	return AssetPair{Token0: split[0], Token1: split[1]}, nil
}

func MustNewAssetPairFromStr(pair string) AssetPair {
	assetPair, err := NewAssetPairFromStr(pair)
	if err != nil {
		panic(err)
	}
	return assetPair
}

// TODO make assetpair a proto type | https://github.com/NibiruChain/nibiru/issues/623

// Name is the name of the pool that corresponds to the two assets on this pair.
func (pair AssetPair) Name() string {
	return PoolNameFromDenoms([]string{pair.Token0, pair.Token1})
}

func (pair AssetPair) PairID() string {
	return pair.Name()
}

func (pair AssetPair) AsString() string {
	return fmt.Sprintf("%s%s%s", pair.Token0, PairSeparator, pair.Token1)
} // Calling this AsString because I'm not seeing a clean way to rewrite
// the proto-generated 'String' methood.

func (pair AssetPair) IsProperOrder() bool {
	return pair.Name() == pair.String()
}

func (pair AssetPair) Inverse() AssetPair {
	return AssetPair{pair.Token1, pair.Token0}
}

func (pair AssetPair) Proper() AssetPair {
	if pair.IsProperOrder() {
		return pair
	} else {
		return pair.Inverse()
	}
}

func (pair AssetPair) GetBaseTokenDenom() string {
	return pair.Token0
}

func (pair AssetPair) GetQuoteTokenDenom() string {
	return pair.Token1
}

func DenomsFromPoolName(pool string) (denoms []string) {
	return strings.Split(pool, ":")
}

// PoolNameFromDenoms returns a sorted string representing a pool of assets
func PoolNameFromDenoms(denoms []string) string {
	sort.Strings(denoms) // alphabetically sort in-place
	return RawPoolNameFromDenoms(denoms)
}

// RawPoolNameFromDenoms returns a string representing a pool of assets in the
// exact order the denoms were given as args
func RawPoolNameFromDenoms(denoms []string) string {
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
		return fmt.Errorf("invalid token1 asset: %w", err)
	}
	if err := sdk.ValidateDenom(pair.Token0); err != nil {
		return fmt.Errorf("invalid token0 asset: %w", err)
	}
	return nil
}

// ----------------------------------- AssetPairs

type AssetPairs []AssetPair

func (pairs AssetPairs) Contains(pair AssetPair) bool {
	for _, element := range pairs {
		if (element.Token0 == pair.Token0) && (element.Token1 == pair.Token1) {
			return true
		}
	}
	return false
}

func (pairs AssetPairs) Strings() []string {
	pairsAsStrings := []string{}
	for _, pair := range pairs {
		pairsAsStrings = append(pairsAsStrings, pair.AsString())
	}
	return pairsAsStrings
}

func (pairs AssetPairs) Validate() error {
	seenPairs := make(map[string]bool)
	for _, pair := range pairs {
		pairID := PoolNameFromDenoms([]string{pair.Token0, pair.Token1})
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

// Contains checks if a token pair is contained within 'Pairs'
func (pairs AssetPairs) ContainsAtIndex(pair AssetPair) (bool, int) {
	for idx, element := range pairs {
		if (element.Token0 == pair.Token0) && (element.Token1 == pair.Token1) {
			return true, idx
		}
	}
	return false, -1
}

func MustNewAssetPairsFromStr(pairStrings []string) (pairs AssetPairs) {
	for _, pairString := range pairStrings {
		pairs = append(pairs, MustNewAssetPairFromStr(pairString))
	}
	return pairs
}
