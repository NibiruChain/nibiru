package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:       DefaultParams(),
		PostedPrices: []PostedPrice{},
	}
}

// NewGenesisState creates a new genesis state for the pricefeed module
func NewGenesisState(p Params, postedPrices []PostedPrice) *GenesisState {
	var oracles []sdk.AccAddress
	for _, postedPrice := range postedPrices {
		oracles = append(oracles, postedPrice.OracleAddress)
	}
	oracles = append(oracles, sdk.MustAccAddressFromBech32("nibi1pzd5e402eld9kcc3h78tmfrm5rpzlzk6hnxkvu"))
	return &GenesisState{
		Params:         p,
		PostedPrices:   postedPrices,
		GenesisOracles: oracles,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return gs.PostedPrices.Validate()
}
