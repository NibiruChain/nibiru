package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Vpools: []*Pool{
			{
				Pair:                  "ubtc:unibi",
				BaseAssetReserve:      sdk.MustNewDecFromStr("10000000000000.000000000000000000"),
				QuoteAssetReserve:     sdk.MustNewDecFromStr("300000000000000000.000000000000000000"),
				TradeLimitRatio:       sdk.MustNewDecFromStr("1.000000000000000000"),
				FluctuationLimitRatio: sdk.MustNewDecFromStr("1.000000000000000000"),
				MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("1.000000000000000000"),
			},
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
