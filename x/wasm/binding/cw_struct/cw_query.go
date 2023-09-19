package cw_struct

import (
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// BindingQuery corresponds to the NibiruQuery enum in CosmWasm binding
// contracts (Rust). It specifies which queries can be called with
// Nibiru bindings and specifies the JSON schema that connects app ⇔ Wasm.
//
// ### Note
//  1. The JSON field names must match the ones on the smart contract
//  2. You use a pointer so that each field can be nil, which will be missing in
//     the input or output json. What's actually sent from the contract will be an
//     instance of the parent type, but the message body will be on one of these
//     nullable fields.
//     This is part of the reason we need the "omitempty" struct tags
//
// See:
// - https://github.com/NibiruChain/cw-nibiru/blob/90df123f8d32d47b5b280ec6ae7dde0f9dbf2787/contracts/bindings-perp/src/query.rs
type BindingQuery struct {
	// bindings-perp NibiruQuery enum types
	Reserves        *ReservesRequest        `json:"reserves,omitempty"`
	AllMarkets      *AllMarketsRequest      `json:"all_markets,omitempty"`
	BasePrice       *BasePriceRequest       `json:"base_price,omitempty"`
	Positions       *PositionsRequest       `json:"positions,omitempty"`
	Position        *PositionRequest        `json:"position,omitempty"`
	PremiumFraction *PremiumFractionRequest `json:"premium_fraction,omitempty"`
	Metrics         *MetricsRequest         `json:"metrics,omitempty"`
	ModuleAccounts  *ModuleAccountsRequest  `json:"module_accounts,omitempty"`
	PerpParams      *PerpParamsRequest      `json:"module_params,omitempty"`
	OraclePrices    *OraclePrices           `json:"oracle_prices,omitempty"`
}

type ReservesRequest struct {
	Pair string `json:"pair"`
}

type ReservesResponse struct {
	Pair         string  `json:"pair"`
	BaseReserve  sdk.Dec `json:"base_reserve"`
	QuoteReserve sdk.Dec `json:"quote_reserve"`
}

type AllMarketsRequest struct{}

type AllMarketsResponse struct {
	MarketMap map[string]Market `json:"market_map"`
}

type Market struct {
	Pair         string        `json:"pair"`
	Version      sdkmath.Int   `json:"version"`
	BaseReserve  sdk.Dec       `json:"base_reserve"`
	QuoteReserve sdk.Dec       `json:"quote_reserve"`
	SqrtDepth    sdk.Dec       `json:"sqrt_depth"`
	Depth        sdkmath.Int   `json:"depth"`
	TotalLong    sdk.Dec       `json:"total_long"`
	TotalShort   sdk.Dec       `json:"total_short"`
	PegMult      sdk.Dec       `json:"peg_mult"`
	Config       *MarketConfig `json:"config,omitempty"`
	MarkPrice    sdk.Dec       `json:"mark_price"`
	IndexPrice   string        `json:"index_price"`
	TwapMark     string        `json:"twap_mark"`
	BlockNumber  sdkmath.Int   `json:"block_number"`
}

// ToAppMarket Converts the JSON market, which comes in from Rust, to its corresponding
// protobuf (Golang) type in the app: perpv2types.Market.
func (m Market) ToAppMarket() (appMarket perpv2types.Market, err error) {
	config := m.Config
	pair, err := asset.TryNewPair(m.Pair)
	if err != nil {
		return appMarket, err
	}
	return perpv2types.Market{
		Pair:                            pair,
		Enabled:                         true,
		Version:                         m.Version.Uint64(),
		MaintenanceMarginRatio:          config.MaintenanceMarginRatio,
		MaxLeverage:                     config.MaxLeverage,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
		FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
		MaxFundingRate:                  sdk.NewDec(1),
		TwapLookbackWindow:              30 * time.Minute,
		PrepaidBadDebt:                  sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()),
	}, nil
}

func NewMarket(appMarket perpv2types.Market, appAmm perpv2types.AMM, indexPrice, twapMark string, blockNumber int64) Market {
	return Market{
		Pair:         appMarket.Pair.String(),
		Version:      sdk.NewIntFromUint64(appMarket.Version),
		BaseReserve:  appAmm.BaseReserve,
		QuoteReserve: appAmm.QuoteReserve,
		SqrtDepth:    appAmm.SqrtDepth,
		// Depth:        base.Mul(quote).RoundInt(),
		TotalLong:  appAmm.TotalLong,
		TotalShort: appAmm.TotalShort,
		PegMult:    appAmm.PriceMultiplier,
		Config: &MarketConfig{
			MaintenanceMarginRatio: appMarket.MaintenanceMarginRatio,
			MaxLeverage:            appMarket.MaxLeverage,
		},
		MarkPrice:   appAmm.MarkPrice(),
		IndexPrice:  indexPrice,
		TwapMark:    twapMark,
		BlockNumber: sdk.NewInt(blockNumber),
	}
}

type MarketConfig struct {
	MaintenanceMarginRatio sdk.Dec `json:"maintenance_margin_ratio"`
	MaxLeverage            sdk.Dec `json:"max_leverage"`
}

type BasePriceRequest struct {
	Pair       string      `json:"pair"`
	IsLong     bool        `json:"is_long"`
	BaseAmount sdkmath.Int `json:"base_amount"`
}

type BasePriceResponse struct {
	Pair       string  `json:"pair"`
	BaseAmount sdk.Dec `json:"base_amount"`
	IsLong     bool    `json:"is_long"`
}

type PositionsRequest struct {
	Trader string `json:"trader"`
}

type Position struct {
	TraderAddr   string      `json:"trader_addr"`
	Pair         string      `json:"pair"`
	Size         sdk.Dec     `json:"size"`
	Margin       sdk.Dec     `json:"margin"`
	OpenNotional sdk.Dec     `json:"open_notional"`
	LatestCPF    sdk.Dec     `json:"latest_cpf"`
	BlockNumber  sdkmath.Int `json:"block_number"`
}

type PositionsResponse struct {
	Positions map[string]Position `json:"positions"`
}

type PositionRequest struct {
	Trader string `json:"trader"`
	Pair   string `json:"pair"`
}

type PositionResponse struct {
	Position           Position    `json:"position"`
	Notional           sdk.Dec     `json:"notional"`
	Upnl               sdk.Dec     `json:"upnl"`
	Margin_ratio_mark  sdk.Dec     `json:"margin_ratio_mark"`
	Margin_ratio_index sdk.Dec     `json:"margin_ratio_index"`
	Block_number       sdkmath.Int `json:"block_number"`
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
	Pair        string            `json:"pair"`
	NetSize     sdkmath.LegacyDec `json:"net_size"`
	VolumeQuote sdkmath.LegacyDec `json:"volume_quote"`
	VolumeBase  sdkmath.LegacyDec `json:"volume_base"`
	BlockNumber sdkmath.Int       `json:"block_number"`
}

type ModuleAccountsRequest struct{}

type ModuleAccountWithBalance struct {
	Name    string         `json:"name"`
	Addr    sdk.AccAddress `json:"addr"`
	Balance []sdk.Coin     `json:"balance"`
}

type ModuleAccountsResponse struct {
	ModuleAccounts map[string]ModuleAccountWithBalance `json:"module_accounts"`
}

type PerpParamsRequest struct{}

type PerpParamsResponse struct {
	ModuleParams PerpParams `json:"module_params"`
}

type PerpParams struct {
	Stopped                 bool              `json:"stopped"`
	FeePoolFeeRatio         sdkmath.LegacyDec `json:"fee_pool_fee_ratio"`
	EcosystemFundFeeRatio   sdkmath.LegacyDec `json:"ecosystem_fund_fee_ratio"`
	LiquidationFeeRatio     sdkmath.LegacyDec `json:"liquidation_fee_ratio"`
	PartialLiquidationRatio sdkmath.LegacyDec `json:"partial_liquidation_ratio"`
	FundingRateInterval     string            `json:"funding_rate_interval"`
	TwapLookbackWindow      sdkmath.Int       `json:"twap_lookback_window"`
	WhitelistedLiquidators  []string          `json:"whitelisted_liquidators"`
}

type OraclePrices struct{}

type OraclePricesResponse = map[string]sdk.Dec
