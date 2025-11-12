package evmtrader

import "math/big"

// OpenTradeParams holds parameters for opening a trade
type OpenTradeParams struct {
	MarketIndex     uint64
	Leverage        uint64
	Long            bool
	CollateralIndex uint64
	TradeType       string // "trade", "limit", or "stop"
	OpenPrice       float64
	TP              *float64 // Optional, nil means omit
	SL              *float64 // Optional, nil means omit
	SlippageP       string   // Default "1" for 100%
	CollateralAmt   *big.Int
}
