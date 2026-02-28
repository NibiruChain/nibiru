package evmtrader_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/sai-trading/services/evmtrader"
	"github.com/stretchr/testify/require"
)

func TestIsLimitOrStopOrder(t *testing.T) {
	tests := []struct {
		name      string
		tradeType string
		expected  bool
	}{
		{
			name:      "market order is not limit or stop",
			tradeType: evmtrader.TradeTypeMarket,
			expected:  false,
		},
		{
			name:      "limit order is limit or stop",
			tradeType: evmtrader.TradeTypeLimit,
			expected:  true,
		},
		{
			name:      "stop order is limit or stop",
			tradeType: evmtrader.TradeTypeStop,
			expected:  true,
		},
		{
			name:      "invalid trade type is not limit or stop",
			tradeType: "invalid",
			expected:  false,
		},
		{
			name:      "empty string is not limit or stop",
			tradeType: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evmtrader.IsLimitOrStopOrder(tt.tradeType)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidTradeType(t *testing.T) {
	tests := []struct {
		name      string
		tradeType string
		expected  bool
	}{
		{
			name:      "market is valid",
			tradeType: evmtrader.TradeTypeMarket,
			expected:  true,
		},
		{
			name:      "limit is valid",
			tradeType: evmtrader.TradeTypeLimit,
			expected:  true,
		},
		{
			name:      "stop is valid",
			tradeType: evmtrader.TradeTypeStop,
			expected:  true,
		},
		{
			name:      "invalid trade type is not valid",
			tradeType: "invalid",
			expected:  false,
		},
		{
			name:      "empty string is not valid",
			tradeType: "",
			expected:  false,
		},
		{
			name:      "uppercase TRADE is not valid",
			tradeType: "TRADE",
			expected:  false,
		},
		{
			name:      "uppercase LIMIT is not valid",
			tradeType: "LIMIT",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evmtrader.IsValidTradeType(tt.tradeType)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTradeTypeConstants(t *testing.T) {
	// Verify the constants have expected values
	require.Equal(t, "trade", evmtrader.TradeTypeMarket)
	require.Equal(t, "limit", evmtrader.TradeTypeLimit)
	require.Equal(t, "stop", evmtrader.TradeTypeStop)
}
