package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/set"
)

// DefaultGenesis returns the default genesis state. This state is used for
// upgrades and for the start of the chain (InitChain).
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultModuleParams(),
		FactoryDenoms: []GenesisDenom{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return err
	}

	seenDenoms := set.New[string]()

	for _, genesisDenom := range gs.GetFactoryDenoms() {
		denom := genesisDenom.GetDenom()
		if seenDenoms.Has(denom) {
			return ErrInvalidGenesis.Wrapf("duplicate denom: %s", denom)
		}
		seenDenoms.Add(denom)

		if err := genesisDenom.Validate(); err != nil {
			return err
		}

		if genesisDenom.AuthorityMetadata.Admin != "" {
			_, err = sdk.AccAddressFromBech32(genesisDenom.AuthorityMetadata.Admin)
			if err != nil {
				return ErrInvalidAuthorityMetadata.Wrapf("Invalid admin address (%s)", err)
			}
		}
	}

	return nil
}
