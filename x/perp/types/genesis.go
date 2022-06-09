package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		PairMetadata: []*PairMetadata{
			{
				Pair: "ubtc:unusd",
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			},
		},
		VaultBalance:         nil,
		PerpEfBalance:        nil,
		FeePoolBalance:       nil,
		Positions:            nil,
		PrepaidBadDebts:      nil,
		WhitelistedAddresses: nil,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
