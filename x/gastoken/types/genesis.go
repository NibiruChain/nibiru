package types

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// DefaultGenesis returns the default txfee genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    Params{},
		Feetokens: []FeeToken{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure. It does not verify that the corresponding pool IDs actually exist.
// This is done in InitGenesis.
func (gs GenesisState) Validate() error {
	seen := make(map[string]struct{})
	for _, feeToken := range gs.Feetokens {
		ok := gethcommon.IsHexAddress(feeToken.Address)
		if !ok {
			return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Address)
		}

		// normalize to checksummed hex for equality checks
		addr := gethcommon.HexToAddress(feeToken.Address).Hex()
		if _, exists := seen[addr]; exists {
			return fmt.Errorf("duplicate fee token address %s", addr)
		}
		seen[addr] = struct{}{}
	}

	return nil
}
