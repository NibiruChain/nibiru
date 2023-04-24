package cw_struct

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// BindingQuery corresponds to the BindingQuery enum in CosmWasm binding
// contracts (Rust). It specifies which queries can be called into the
// Nibiru bindings, and describes their JSON schema for connecting app ⇔ Wasm.
//
// ### Note
// 1. The JSON field names must match the ones on the smart contract
type BindingQuery struct {
	Reserves        *ReservesRequest        `json:"reserves,omitempty"`
	AllMarkets      *AllMarketsRequest      `json:"all_markets,omitempty"`
	BasePrice       *BasePriceRequest       `json:"base_price,omitempty"`
	Positions       *PositionsRequest       `json:"positions,omitempty"`
	Position        *PositionRequest        `json:"position,omitempty"`
	PremiumFraction *PremiumFractionRequest `json:"premium_fraction,omitempty"`
	Metrics         *MetricsRequest         `json:"metrics,omitempty"`
	ModuleAccounts  *ModuleAccountsRequest  `json:"module_accounts,omitempty"`
	PerpParams      *PerpParamsRequest      `json:"module_params,omitempty"`
}

type ReservesRequest struct {
	Pair string `json:"pair"`
}

type ReservesResponse struct {
	Pair         string  `json:"pair"`
	BaseReserve  sdk.Dec `json:"base_reserve"`
	QuoteReserve sdk.Dec `json:"quote_reserve"`
}

type AllMarketsRequest struct {
}

type AllMarketsResponse struct {
	MarketMap map[string]Market `json:"market_map"`
}

type Market struct {
	Pair         string        `json:"pair"`
	BaseReserve  sdk.Dec       `json:"base_reserve"`
	QuoteReserve sdk.Dec       `json:"quote_reserve"`
	SqrtDepth    sdk.Dec       `json:"sqrt_depth"`
	Depth        sdk.Int       `json:"depth"`
	Bias         sdk.Dec       `json:"bias"`
	PegMult      sdk.Dec       `json:"peg_mult"`
	Config       *MarketConfig `json:"config,omitempty"`
	MarkPrice    sdk.Dec       `json:"mark_price"`
	IndexPrice   string        `json:"index_price"`
	TwapMark     string        `json:"twap_mark"`
	BlockNumber  sdk.Int       `json:"block_number"`
}

// Converts the JSON market, which comes in from Rust, to its corresponding
// protobuf (Golang) type in the app: perpammtypes.Market.
func (m Market) ToAppMarket() (appMarket perpammtypes.Market, err error) {
	config := m.Config
	pair, err := asset.TryNewPair(m.Pair)
	if err != nil {
		return appMarket, err
	}
	return perpammtypes.NewMarket(perpammtypes.ArgsNewMarket{
		Pair:          pair,
		BaseReserves:  m.BaseReserve,
		QuoteReserves: m.QuoteReserve,
		Config: &perpammtypes.MarketConfig{
			TradeLimitRatio:        config.TradeLimitRatio,
			FluctuationLimitRatio:  config.FluctLimitRatio,
			MaxOracleSpreadRatio:   config.MaxOracleSpreadRatio,
			MaintenanceMarginRatio: config.MaintenanceMarginRatio,
			MaxLeverage:            config.MaxLeverage,
		},
		Bias:          m.Bias,
		PegMultiplier: m.PegMult,
	}), nil
}

func NewMarket(appMarket perpammtypes.Market, indexPrice, twapMark string, blockNumber int64) Market {
	base := appMarket.BaseReserve
	quote := appMarket.QuoteReserve
	return Market{
		Pair:         appMarket.Pair.String(),
		BaseReserve:  base,
		QuoteReserve: quote,
		SqrtDepth:    appMarket.SqrtDepth,
		Depth:        base.Mul(quote).RoundInt(),
		Bias:         appMarket.Bias,
		PegMult:      appMarket.PegMultiplier,
		Config:       NewMarketConfig(appMarket.Config),
		MarkPrice:    appMarket.GetMarkPrice(),
		IndexPrice:   indexPrice,
		TwapMark:     twapMark,
		BlockNumber:  sdk.NewInt(blockNumber),
	}
}

type MarketConfig struct {
	TradeLimitRatio        sdk.Dec `json:"trade_limit_ratio"`
	FluctLimitRatio        sdk.Dec `json:"fluct_limit_ratio"`
	MaxOracleSpreadRatio   sdk.Dec `json:"max_oracle_spread_ratio"`
	MaintenanceMarginRatio sdk.Dec `json:"maintenance_margin_ratio"`
	MaxLeverage            sdk.Dec `json:"max_leverage"`
}

func NewMarketConfig(
	appMarketConfig perpammtypes.MarketConfig,
) *MarketConfig {
	return &MarketConfig{
		TradeLimitRatio:        appMarketConfig.TradeLimitRatio,
		FluctLimitRatio:        appMarketConfig.FluctuationLimitRatio,
		MaxOracleSpreadRatio:   appMarketConfig.MaxOracleSpreadRatio,
		MaintenanceMarginRatio: appMarketConfig.MaintenanceMarginRatio,
		MaxLeverage:            appMarketConfig.MaxLeverage,
	}
}

type BasePriceRequest struct {
	Pair       string  `json:"pair"`
	IsLong     bool    `json:"is_long"`
	BaseAmount sdk.Int `json:"base_amount"`
}

type BasePriceResponse struct {
	Pair        string  `json:"pair"`
	BaseAmount  sdk.Dec `json:"base_amount"`
	QuoteAmount sdk.Dec `json:"quote_amount"`
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

type PremiumFractionResponse struct {
	Pair             string  `json:"pair"`
	CPF              sdk.Dec `json:"cpf"`
	EstimatedNextCPF sdk.Dec `json:"estimated_next_cpf"`
}

type MetricsRequest struct {
	Pair string `json:"pair"`
}

type MetricsResponse struct {
	Metrics Metrics `json:"metrics"`
}

type Metrics struct {
	Pair        string  `json:"pair"`
	NetSize     sdk.Dec `json:"net_size"`
	VolumeQuote sdk.Dec `json:"volume_quote"`
	VolumeBase  sdk.Dec `json:"volume_base"`
	BlockNumber sdk.Int `json:"block_number"`
}

type ModuleAccountsRequest struct {
}

type ModuleAccountWithBalance struct {
	Name    string         `json:"name"`
	Addr    sdk.AccAddress `json:"addr"`
	Balance []sdk.Coin     `json:"balance"`
}

type ModuleAccountsResponse struct {
	ModuleAccounts map[string]ModuleAccountWithBalance `json:"module_accounts"`
}

type PerpParamsRequest struct {
}

type PerpParamsResponse struct {
	ModuleParams PerpParams `json:"module_params"`
}

type PerpParams struct {
	Stopped                 bool     `json:"stopped"`
	FeePoolFeeRatio         sdk.Dec  `json:"fee_pool_fee_ratio"`
	EcosystemFundFeeRatio   sdk.Dec  `json:"ecosystem_fund_fee_ratio"`
	LiquidationFeeRatio     sdk.Dec  `json:"liquidation_fee_ratio"`
	PartialLiquidationRatio sdk.Dec  `json:"partial_liquidation_ratio"`
	FundingRateInterval     string   `json:"funding_rate_interval"`
	TwapLookbackWindow      sdk.Int  `json:"twap_lookback_window"`
	WhitelistedLiquidators  []string `json:"whitelisted_liquidators"`
}
