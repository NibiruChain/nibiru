package evmtrader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatWasmDecimal(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{name: "large price", input: 69000, expected: "69000"},
		{name: "nibi usd oracle rate", input: 0.0014689879917464995, expected: "0.0014689879917465"},
		{name: "btc scale", input: 65028.15, expected: "65028.15"},
		{name: "integer", input: 1, expected: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWasmDecimal(tt.input)
			require.Equal(t, tt.expected, got)
			if strings.Contains(got, ".") {
				parts := strings.SplitN(got, ".", 2)
				require.LessOrEqual(t, len(parts[1]), maxWasmDecimalPlaces)
			}
		})
	}
}

func TestFormatPriceForLog(t *testing.T) {
	require.Equal(t, "$65028.15", formatPriceForLog(65028.15))
	require.Equal(t, "$0.0014689879917465", formatPriceForLog(0.0014689879917464995))
}
