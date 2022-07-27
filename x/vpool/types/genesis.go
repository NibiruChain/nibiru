package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// DefaultGenesis returns the default vpool genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		// TODO(https://github.com/NibiruChain/nibiru/issues/747): replace with reasonable defaults for mainnet
		Vpools: []*Pool{
			{
				Pair:                   common.PairBTCStable,
				QuoteAssetReserve:      sdk.NewDec(1e9 * 1e6),           // 1 billion NUSD
				BaseAssetReserve:       sdk.NewDec(50_000 * 1e6),        // 50k BTC
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.1"),    // 10%
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),    // 10%
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),    // 10%
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"), // 6.25%
			},
			{
				Pair:                   common.PairETHStable,
				QuoteAssetReserve:      sdk.NewDec(1e9 * 1e6),           // 1 billion NUSD
				BaseAssetReserve:       sdk.NewDec(666_666 * 1e6),       // 666k ETH
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.1"),    // 10%
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),    // 10%
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),    // 10%
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"), // 6.25%
			},
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
