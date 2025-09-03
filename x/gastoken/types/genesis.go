package types

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// DefaultGenesis returns the default txfee genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: Params{
			UniswapV3SwapRouterAddress: gethcommon.Address{}.String(),
			UniswapV3QuoterAddress:     gethcommon.Address{}.String(),
			WnibiAddress:               gethcommon.Address{}.String(),
		},
		Feetokens: []FeeToken{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure. It does not verify that the corresponding pool IDs actually exist.
// This is done in InitGenesis.
func (gs GenesisState) Validate() error {
	seen := make(map[string]struct{})
	for _, feeToken := range gs.Feetokens {
		ok := gethcommon.IsHexAddress(feeToken.Erc20Address)
		if !ok {
			return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Erc20Address)
		}

		// normalize to checksummed hex for equality checks
		addr := gethcommon.HexToAddress(feeToken.Erc20Address).Hex()
		if _, exists := seen[addr]; exists {
			return fmt.Errorf("duplicate fee token address %s", addr)
		}
		seen[addr] = struct{}{}
	}

	return nil
}
