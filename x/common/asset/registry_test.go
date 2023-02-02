package asset

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/denoms"
)

func TestIsSupportedPair(t *testing.T) {
	for base := range Registry {
		for quote := range Registry[base] {
			require.Truef(t, Registry.IsSupportedPair(base, quote), "%s:%s should be supported", base, quote)
		}
	}

	t.Log("test an unsupported pair")
	require.False(t, Registry.IsSupportedPair(denoms.ATOM, denoms.OSMO))
}

func TestPair(t *testing.T) {
	for base := range Registry {
		for quote := range Registry[base] {
			require.Equal(t, NewPair(base, quote), Registry.Pair(base, quote))
		}
	}

	t.Log("test an unsupported pair")
	require.Equal(t, Pair(""), Registry.Pair(denoms.ATOM, denoms.OSMO))

	t.Log("test an unsupported base asset")
	require.Equal(t, Pair(""), Registry.Pair("unsuported_denom", denoms.USDC))

	t.Log("test an unsupported quote asset")
	require.Equal(t, Pair(""), Registry.Pair(denoms.ATOM, "unsupported_denom"))
}

func TestBaseDenoms(t *testing.T) {
	for base := range Registry {
		require.Contains(t, Registry.BaseDenoms(), base)
	}
}

func TestIsSupportedBaseDenom(t *testing.T) {
	for base := range Registry {
		require.True(t, Registry.IsSupportedBaseDenom(base))
	}
	require.False(t, Registry.IsSupportedBaseDenom("unsupported_denom"))
}

func TestQuoteDenoms(t *testing.T) {
	for base := range Registry {
		for quote := range Registry[base] {
			require.True(t, Registry.QuoteDenoms().Has(quote))
		}
	}
}

func TestIsSupportedQuoteDenom(t *testing.T) {
	for base := range Registry {
		for quote := range Registry[base] {
			require.True(t, Registry.IsSupportedQuoteDenom(quote))
		}
	}

	require.False(t, Registry.IsSupportedQuoteDenom("unsupported_denom"))
}

func TestIsSupportedDenom(t *testing.T) {
	for base := range Registry.BaseDenoms() {
		require.True(t, Registry.IsSupportedDenom(base))
	}

	for quote := range Registry.QuoteDenoms() {
		require.True(t, Registry.IsSupportedDenom(quote))
	}

	t.Log("test an unsupported denom")
	require.False(t, Registry.IsSupportedDenom("unsupported_denom"))
}
