package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	// TODO remove after demo
	return &GenesisState{
		Params: DefaultParams(),
		Vpools: []*Pool{
			{
				Pair:                  "ubtc:unibi",
				BaseAssetReserve:      sdk.MustNewDecFromStr("10000000"),
				QuoteAssetReserve:     sdk.MustNewDecFromStr("60000000000"),
				TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
			},
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
