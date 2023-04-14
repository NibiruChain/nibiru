package cw_struct

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BindingQuery corresponds to the BindingQuery enum in CosmWasm binding
// contracts (Rust). It specifies which queries can be called into the
// Nibiru bindings, and describes their JSON schema for connecting app ⇔ Wasm.
type BindingQuery struct {
	Reserves        *ReservesRequest
	AllMarkets      *AllMarketsRequest
	BasePrice       *BasePriceRequest
	Positions       *PositionsRequest
	Position        *PositionRequest
	PremiumFraction *PremiumFractionRequest
	Metrics         *MetricsRequest
	ModuleAccounts  *ModuleAccountsRequest
	PerpParams      *PerpParamsRequest
}

type ReservesRequest struct {
	Pair string `json:"pair"`
}

// TODO
type ReservesResponse struct {
	Pair         string  `json:"pair"`
	BaseReserve  sdk.Dec `json:"base_reserve"`
	QuoteReserve sdk.Dec `json:"quote_reserve"`
}

type AllMarketsRequest struct {
}

type AllMarketsResponse struct {
	MarketMap map[string]Market `json:"markets"`
}

type Market struct {
	Pair         string       `json:"pair"`
	BaseReserve  sdk.Dec      `json:"base_reserve"`
	QuoteReserve sdk.Dec      `json:"quote_reserve"`
	SqrtDepth    sdk.Dec      `json:"sqrt_depth"`
	Depth        sdk.Int      `json:"depth"`
	Bias         sdk.Dec      `json:"bias"`
	PegMult      sdk.Dec      `json:"pegmult"`
	Config       MarketConfig `json:"config"`
	MarkPrice    sdk.Dec      `json:"mark_price"`
	IndexPrice   string       `json:"index_price"`
	TwapMark     string       `json:"twap_mark"`
	BlockNumber  int64        `json:"block_number"`
}

type MarketConfig struct {
	TradeLimitRatio        sdk.Dec `json:"trade_limit_ratio"`
	FluctLimitRatio        sdk.Dec `json:"fluct_limit_ratio"`
	MaxOracleSpreadRatio   sdk.Dec `json:"max_oracle_spread_ratio"`
	MaintenanceMarginRatio sdk.Dec `json:"maintenance_margin_ratio"`
	MaxLeverage            sdk.Dec `json:"max_leverage"`
}

type BasePriceRequest struct {
	Pair       string  `json:"pair"`
	IsLong     bool    `json:"is_long"`
	BaseAmount sdk.Int `json:"base_amount"`
}

type BasePriceResponse struct {
	BaseAmount  sdk.Int `json:"base_amount"`
	QuoteAmount sdk.Int `json:"quote_amount"`
	IsLong      bool    `json:"is_long"`
}

type PositionsRequest struct {
	Trader string `json:"trader"`
}

// TODO impl
type PositionsResponse struct {
	Trader string `json:"trader"`
}

type PositionRequest struct {
	Trader string `json:"trader"`
	Pair   string `json:"pair"`
}

// TODO impl
type PositionResponse struct {
	Trader string `json:"trader"`
}

type PremiumFractionRequest struct {
	Pair string `json:"pair"`
}

type MetricsRequest struct {
	Pair string `json:"pair"`
}

type ModuleAccountsRequest struct {
}

type PerpParamsRequest struct {
}
