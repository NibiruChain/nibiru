package common

import "github.com/NibiruChain/nibiru/x/common/denoms"

type assetRegistry map[string]StringSet

var AssetRegistry assetRegistry

func init() {
	// map of base asset to supported quote assets
	// quote assets are usually stables
	AssetRegistry = map[string]StringSet{
		denoms.DenomBTC:  NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomETH:  NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomNIBI: NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomATOM: NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomOSMO: NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomAVAX: NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomSOL:  NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomBNB:  NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomADA:  NewStringSet(denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT),
		denoms.DenomNUSD: NewStringSet(denoms.DenomUSD, denoms.DenomUSDC),
		denoms.DenomUSDC: NewStringSet(denoms.DenomUSD, denoms.DenomNUSD),
		denoms.DenomUSDT: NewStringSet(denoms.DenomUSD, denoms.DenomNUSD, denoms.DenomUSDC),
	}
}

func (r assetRegistry) Pair(base string, quote string) AssetPair {
	for q := range r[base] {
		if q == quote {
			return NewAssetPair(string(base), string(quote))
		}
	}

	return ""
}

// Returns all supported base denoms
func (r assetRegistry) BaseDenoms() StringSet {
	baseSet := make(StringSet)
	for d := range r {
		baseSet.Add(d)
	}
	return baseSet
}

// Returns all supported quote denoms
func (r assetRegistry) QuoteDenoms() StringSet {
	quoteSet := make(StringSet)
	for base := range r {
		for q := range r[base] {
			quoteSet.Add(q)
		}
	}
	return quoteSet
}

// Checks if the provided denom is a supported base denom
func (r assetRegistry) IsSupportedBaseDenom(denom string) bool {
	_, ok := r[denom]
	return ok
}

// Checks if the provided denom is a supported quote denom
func (r assetRegistry) IsSupportedQuoteDenom(denom string) bool {
	return r.QuoteDenoms().Has(denom)
}

// Checks if the provided denom is a supported denom
func (r assetRegistry) IsSupportedDenom(denom string) bool {
	return r.IsSupportedBaseDenom(string(denom)) || r.IsSupportedQuoteDenom(string(denom))
}

// Checks if the provided base and quote denoms are a supported pair
func (r assetRegistry) IsSupportedPair(base string, quote string) bool {
	return r.IsSupportedBaseDenom(base) && r.IsSupportedQuoteDenom(quote)
}
