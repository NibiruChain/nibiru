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
		Feetoken:  FeeToken{},
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

	ok := gethcommon.IsHexAddress(gs.Feetoken.Address)
	if !ok {
		return fmt.Errorf("invalid fee token address %s: must be a valid hex address", gs.Feetoken.Address)
	}

	return nil
}
