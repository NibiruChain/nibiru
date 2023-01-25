package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSupportedPair(t *testing.T) {
	for base := range AssetRegistry {
		for _, quote := range AssetRegistry[base] {
			require.Truef(t, AssetRegistry.IsSupportedPair(base, quote), "%s:%s should be supported", base, quote)
		}
	}

	t.Log("test an unsupported pair")
	require.False(t, AssetRegistry.IsSupportedPair(DenomATOM, DenomOSMO))
}

func TestPair(t *testing.T) {
	for base := range AssetRegistry {
		for _, quote := range AssetRegistry[base] {
			require.Equal(t, NewAssetPair(base, quote), AssetRegistry.Pair(base, quote))
		}
	}

	t.Log("test an unsupported pair")
	require.Equal(t, AssetPair(""), AssetRegistry.Pair(DenomATOM, DenomOSMO))

	t.Log("test an unsupported base asset")
	require.Equal(t, AssetPair(""), AssetRegistry.Pair("unsuported_denom", DenomUSDC))

	t.Log("test an unsupported quote asset")
	require.Equal(t, AssetPair(""), AssetRegistry.Pair(DenomATOM, "unsupported_denom"))
}

func TestBaseDenoms(t *testing.T) {
	for base := range AssetRegistry {
		require.Contains(t, AssetRegistry.BaseDenoms(), base)
	}
}

func TestIsSupportedBaseDenom(t *testing.T) {
	for base := range AssetRegistry {
		require.True(t, AssetRegistry.IsSupportedBaseDenom(base))
	}
	require.False(t, AssetRegistry.IsSupportedBaseDenom("unsupported_denom"))
}

func TestQuoteDenoms(t *testing.T) {
	for base := range AssetRegistry {
		for _, quote := range AssetRegistry[base] {
			require.Contains(t, AssetRegistry.QuoteDenoms(), quote)
		}
	}
}

func TestIsSupportedQuoteDenom(t *testing.T) {
	for base := range AssetRegistry {
		for _, quote := range AssetRegistry[base] {
			require.True(t, AssetRegistry.IsSupportedQuoteDenom(quote))
		}
	}

	require.False(t, AssetRegistry.IsSupportedQuoteDenom("unsupported_denom"))
}

func TestIsSupportedDenom(t *testing.T) {
	for _, base := range AssetRegistry.BaseDenoms() {
		require.True(t, AssetRegistry.IsSupportedDenom(base))
	}

	for _, quote := range AssetRegistry.QuoteDenoms() {
		require.True(t, AssetRegistry.IsSupportedDenom(quote))
	}

	t.Log("test an unsupported denom")
	require.False(t, AssetRegistry.IsSupportedDenom("unsupported_denom"))
}
