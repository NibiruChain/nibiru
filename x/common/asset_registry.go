package common

import "github.com/NibiruChain/nibiru/x/common/denoms"

type assetRegistry map[string][]string

var AssetRegistry assetRegistry

func init() {
	// map of base asset to supported quote assets
	// quote assets are usually stables
	AssetRegistry = map[string][]string{
		denoms.DenomBTC:  {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomETH:  {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomNIBI: {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomATOM: {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomOSMO: {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomAVAX: {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomSOL:  {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomBNB:  {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomADA:  {denoms.DenomUSDC, denoms.DenomNUSD, denoms.DenomUSD, denoms.DenomUSDT},
		denoms.DenomNUSD: {denoms.DenomUSD, denoms.DenomUSDC},
		denoms.DenomUSDC: {denoms.DenomUSD, denoms.DenomNUSD},
		denoms.DenomUSDT: {denoms.DenomUSD, denoms.DenomNUSD, denoms.DenomUSDC},
	}
}

func (r assetRegistry) Pair(base string, quote string) AssetPair {
	for _, q := range r[base] {
		if q == quote {
			return NewAssetPair(string(base), string(quote))
		}
	}

	return ""
}

// Returns all supported base denoms
func (r assetRegistry) BaseDenoms() []string {
	var denoms []string
	for d := range r {
		denoms = append(denoms, d)
	}
	return denoms
}

// Returns all supported quote denoms
func (r assetRegistry) QuoteDenoms() []string {
	var denoms []string
	for _, q := range r {
		denoms = append(denoms, q...)
	}
	return denoms
}

// Checks if the provided denom is a supportedd base denom
func (r assetRegistry) IsSupportedBaseDenom(denom string) bool {
	_, ok := r[denom]
	return ok
}

// Checks if the provided denom is a supported quote denom
func (r assetRegistry) IsSupportedQuoteDenom(denom string) bool {
	for _, q := range r {
		for _, d := range q {
			if d == denom {
				return true
			}
		}
	}
	return false
}

// Checks if the provided denom is a supported denom
func (r assetRegistry) IsSupportedDenom(denom string) bool {
	return r.IsSupportedBaseDenom(string(denom)) || r.IsSupportedQuoteDenom(string(denom))
}

// Checks if the provided base and quote denoms are a supported pair
func (r assetRegistry) IsSupportedPair(base string, quote string) bool {
	return r.IsSupportedBaseDenom(base) && r.IsSupportedQuoteDenom(quote)
}
