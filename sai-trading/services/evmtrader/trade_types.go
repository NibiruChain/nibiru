package evmtrader

// Trade type constants
const (
	TradeTypeMarket = "trade"
	TradeTypeLimit  = "limit"
	TradeTypeStop   = "stop"
)

// IsLimitOrStopOrder returns true if the trade type is a limit or stop order.
func IsLimitOrStopOrder(tradeType string) bool {
	return tradeType == TradeTypeLimit || tradeType == TradeTypeStop
}

// IsValidTradeType returns true if the trade type is valid.
func IsValidTradeType(tradeType string) bool {
	return tradeType == TradeTypeMarket ||
		tradeType == TradeTypeLimit ||
		tradeType == TradeTypeStop
}

// isLimitOrStopOrder is an internal helper that calls IsLimitOrStopOrder.
// This is provided for backward compatibility with internal code.
func isLimitOrStopOrder(tradeType string) bool {
	return IsLimitOrStopOrder(tradeType)
}
