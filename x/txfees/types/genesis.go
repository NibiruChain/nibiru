package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

// DefaultGenesis returns the default txfee genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Basedenom: sdk.DefaultBondDenom,
		Feetokens: []FeeToken{},
		Params:    DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure. It does not verify that the corresponding pool IDs actually exist.
// This is done in InitGenesis.
func (gs GenesisState) Validate() error {
	err := sdk.ValidateDenom(gs.Basedenom)
	if err != nil {
		return err
	}

	for _, feeToken := range gs.Feetokens {
		ok := gethcommon.IsHexAddress(feeToken.Denom)
		if !ok {
			return fmt.Errorf("invalid fee token denom %s: must be a valid hex address", feeToken.Denom)
		}
	}

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}
