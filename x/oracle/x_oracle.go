package oracle

import _ "embed"

// XOracleAdapterWasm is the Sai x-oracle adapter contract fixture.
//
// - Source Repo: https://github.com/NibiruChain/sai-perps
// - Path: sai-perps/contracts/x-oracle
// - Build: Wasm optimized build artifact.
//
//go:embed x_oracle.wasm
var XOracleAdapterWasm []byte

type XOracleAdapterMode struct {
	Fixture *struct{}                `json:"fixture,omitempty"`
	Proxy   *XOracleAdapterProxyMode `json:"proxy,omitempty"`
}

type XOracleAdapterProxyMode struct {
	SaiOracleAddr string `json:"sai_oracle_addr"`
}

func XOracleAdapterFixtureMode() XOracleAdapterMode {
	return XOracleAdapterMode{Fixture: &struct{}{}}
}

func XOracleAdapterProxyModeOf(saiOracleAddr string) XOracleAdapterMode {
	return XOracleAdapterMode{
		Proxy: &XOracleAdapterProxyMode{SaiOracleAddr: saiOracleAddr},
	}
}

type XOracleAdapterInstantiateMsg struct {
	Owner          string                        `json:"owner,omitempty"`
	Mode           XOracleAdapterMode            `json:"mode"`
	LegacyMappings []XOracleAdapterLegacyMapping `json:"legacy_mappings"`
}

type XOracleAdapterLegacyMapping struct {
	Symbol     string `json:"symbol"`
	TokenIndex uint16 `json:"token_index"`
}

type XOracleAdapterQueryMsg struct {
	GetPrice            *XOracleAdapterGetPriceQuery            `json:"get_price,omitempty"`
	LegacyExchangeRate  *XOracleAdapterLegacyExchangeRateQuery  `json:"legacy_exchange_rate,omitempty"`
	LegacyExchangeRates *XOracleAdapterLegacyExchangeRatesQuery `json:"legacy_exchange_rates,omitempty"`
	Config              *struct{}                               `json:"config,omitempty"`
}

type XOracleAdapterGetPriceQuery struct {
	Index uint16 `json:"index"`
}

type XOracleAdapterLegacyExchangeRateQuery struct {
	Symbol string `json:"symbol"`
}

type XOracleAdapterLegacyExchangeRatesQuery struct{}

type XOracleAdapterPriceResp struct {
	Price             string  `json:"price"`
	LastOracleAddress *string `json:"last_oracle_address"`
	LastUpdateTime    *uint64 `json:"last_update_time"`
}

type XOracleAdapterLegacyExchangeRateResp struct {
	Symbol            string  `json:"symbol"`
	TokenIndex        uint16  `json:"token_index"`
	Price18           string  `json:"price_18"`
	PriceDecimal      string  `json:"price_decimal"`
	Decimals          uint8   `json:"decimals"`
	UpdateTimeSeconds *uint64 `json:"update_time_seconds"`
}

type XOracleAdapterLegacyExchangeRatesResp struct {
	Rates []XOracleAdapterLegacyExchangeRateResp `json:"rates"`
}

type XOracleAdapterConfigResp struct {
	Mode           map[string]any                `json:"mode"`
	LegacyMappings []XOracleAdapterLegacyMapping `json:"legacy_mappings"`
}
