package types

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// DefaultGenesis returns the default txfee genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Feetokens: []FeeToken{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure. It does not verify that the corresponding pool IDs actually exist.
// This is done in InitGenesis.
func (gs GenesisState) Validate() error {
	for _, feeToken := range gs.Feetokens {
		ok := gethcommon.IsHexAddress(feeToken.Address)
		if !ok {
			return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Address)
		}
	}

	return nil
}
