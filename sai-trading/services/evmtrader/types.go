package evmtrader

import "math/big"

// OpenTradeParams holds parameters for opening a trade
type OpenTradeParams struct {
	MarketIndex     uint64
	Leverage        uint64
	Long            bool
	CollateralIndex uint64
	TradeType       string   // "trade", "limit", or "stop"
	OpenPrice       *float64 // Optional, nil means omit (for market orders, contract uses execution price)
	TP              *float64 // Optional, nil means omit
	SL              *float64 // Optional, nil means omit
	SlippageP       string   // Default "1" for 100%
	CollateralAmt   *big.Int
}

// MarketInfo holds information about a market
type MarketInfo struct {
	Index       uint64
	BaseToken   *uint64
	QuoteToken  *uint64
	MaxOI       *string
	FeePerBlock *string
}

// Trade holds information about a user's trade/position
type Trade struct {
	User              string  `json:"user"`
	MarketIndex       string  `json:"market_index"`     // "MarketIndex(0)"
	UserTradeIndex    string  `json:"user_trade_index"` // "UserTradeIndex(0)"
	Leverage          string  `json:"leverage"`
	Long              bool    `json:"long"`
	IsOpen            bool    `json:"is_open"`
	CollateralIndex   string  `json:"collateral_index"` // "TokenIndex(1)"
	TradeType         string  `json:"trade_type"`
	CollateralAmount  string  `json:"collateral_amount"`
	OpenPrice         string  `json:"open_price"`
	OpenCollateralAmt string  `json:"open_collateral_amount"`
	TP                string  `json:"tp"`
	SL                *string `json:"sl,omitempty"`
	IsEvmOrigin       bool    `json:"is_evm_origin"`
}

// ParsedTrade holds parsed trade information with numeric indices
type ParsedTrade struct {
	User              string
	MarketIndex       uint64
	UserTradeIndex    uint64
	Leverage          string
	Long              bool
	IsOpen            bool
	CollateralIndex   uint64
	TradeType         string
	CollateralAmount  string
	OpenPrice         string
	OpenCollateralAmt string
	TP                string
	SL                *string
	IsEvmOrigin       bool
}
