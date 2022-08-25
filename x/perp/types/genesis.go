package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:               DefaultParams(),
		VaultBalance:         []sdk.Coin(nil),
		PerpEfBalance:        []sdk.Coin(nil),
		FeePoolBalance:       []sdk.Coin(nil),
		PairMetadata:         []*PairMetadata(nil),
		Positions:            []*Position(nil),
		PrepaidBadDebts:      []*PrepaidBadDebt(nil),
		WhitelistedAddresses: []string(nil),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for i, pos := range gs.Positions {
		if err := pos.Validate(); err != nil {
			return fmt.Errorf("malformed genesis position %s at index %d: %w", pos, i, err)
		}
	}
	return gs.Params.Validate()
}
