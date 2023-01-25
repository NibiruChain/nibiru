package common

type assetRegistry map[string][]string

var AssetRegistry assetRegistry

func init() {
	// map of base asset to supported quote assets
	// quote assets are usually stables
	AssetRegistry = map[string][]string{
		DenomBTC:  {DenomUSDC, DenomNUSD, DenomUSD, DenomUSDT},
		DenomETH:  {DenomUSDC, DenomNUSD, DenomUSD, DenomUSDT},
		DenomNIBI: {DenomUSDC, DenomNUSD, DenomUSD, DenomUSDT},
		DenomATOM: {DenomUSDC, DenomNUSD, DenomUSD, DenomUSDT},
		DenomOSMO: {DenomUSDC, DenomNUSD, DenomUSD, DenomUSDT},
		DenomNUSD: {DenomUSD},
		DenomUSDC: {DenomUSD},
		DenomUSDT: {DenomUSD},
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
