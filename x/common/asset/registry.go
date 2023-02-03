package asset

import (
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/set"
)

type registry map[string]set.Set[string]

var Registry registry

func init() {
	// map of base asset to supported quote assets
	// quote assets are usually stables
	Registry = map[string]set.Set[string]{
		denoms.BTC:  set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.ETH:  set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.NIBI: set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.ATOM: set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.OSMO: set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.AVAX: set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.SOL:  set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.BNB:  set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.ADA:  set.New(denoms.USDC, denoms.NUSD, denoms.USD, denoms.USDT),
		denoms.NUSD: set.New(denoms.USD, denoms.USDC),
		denoms.USDC: set.New(denoms.USD, denoms.NUSD),
		denoms.USDT: set.New(denoms.USD, denoms.NUSD, denoms.USDC),
	}
}

func (r registry) Pair(base string, quote string) Pair {
	for q := range r[base] {
		if q == quote {
			return NewPair(string(base), string(quote))
		}
	}

	return ""
}

// Returns all supported base denoms
func (r registry) BaseDenoms() set.Set[string] {
	baseSet := make(set.Set[string])
	for d := range r {
		baseSet.Add(d)
	}
	return baseSet
}

// Returns all supported quote denoms
func (r registry) QuoteDenoms() set.Set[string] {
	quoteSet := make(set.Set[string])
	for base := range r {
		for q := range r[base] {
			quoteSet.Add(q)
		}
	}
	return quoteSet
}

// Checks if the provided denom is a supported base denom
func (r registry) IsSupportedBaseDenom(denom string) bool {
	_, ok := r[denom]
	return ok
}

// Checks if the provided denom is a supported quote denom
func (r registry) IsSupportedQuoteDenom(denom string) bool {
	return r.QuoteDenoms().Has(denom)
}

// Checks if the provided denom is a supported denom
func (r registry) IsSupportedDenom(denom string) bool {
	return r.IsSupportedBaseDenom(string(denom)) || r.IsSupportedQuoteDenom(string(denom))
}

// Checks if the provided base and quote denoms are a supported pair
func (r registry) IsSupportedPair(base string, quote string) bool {
	return r.IsSupportedBaseDenom(base) && r.IsSupportedQuoteDenom(quote)
}
