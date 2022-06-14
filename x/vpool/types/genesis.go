package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default vpool genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Vpools: []*Pool{
			{
				Pair:                  "ubtc:unusd",
				BaseAssetReserve:      sdk.NewDec(1_000),
				QuoteAssetReserve:     sdk.NewDec(50_000_000),
				TradeLimitRatio:       sdk.OneDec(),
				FluctuationLimitRatio: sdk.OneDec(),
				MaxOracleSpreadRatio:  sdk.OneDec(),
			},
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
