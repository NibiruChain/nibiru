package evmtrader

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// maxWasmDecimalPlaces is the maximum fractional digits accepted by CosmWasm Decimal.
const maxWasmDecimalPlaces = 18

// formatWasmDecimal formats a float for perp contract Decimal fields (max 18 fractional digits).
func formatWasmDecimal(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	// 15 significant digits avoids float64 noise; cap fractional digits for CosmWasm Decimal.
	s := strconv.FormatFloat(v, 'g', 15, 64)
	if strings.ContainsAny(s, "eE") {
		s = strconv.FormatFloat(v, 'f', maxWasmDecimalPlaces, 64)
	}
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart := s[:dot]
		frac := s[dot+1:]
		if len(frac) > maxWasmDecimalPlaces {
			frac = frac[:maxWasmDecimalPlaces]
		}
		frac = strings.TrimRight(frac, "0")
		if frac == "" {
			if intPart == "" || intPart == "-" {
				return "0"
			}
			return intPart
		}
		return intPart + "." + frac
	}
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// formatPriceForLog formats a price for human-readable logs (handles sub-dollar assets).
func formatPriceForLog(price float64) string {
	if price >= 1 {
		return fmt.Sprintf("$%.2f", price)
	}
	return fmt.Sprintf("$%s", formatWasmDecimal(price))
}
