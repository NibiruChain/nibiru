package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/common/set"
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
			return ErrInvalidGenesis.Wrap(err.Error())
		}

		if admin := genesisDenom.AuthorityMetadata.Admin; admin != "" {
			_, err = sdk.AccAddressFromBech32(admin)
			if err != nil {
				return fmt.Errorf("%w: %s: admin address (%s): %s",
					ErrInvalidGenesis, ErrInvalidAdmin, admin, err,
				)
			}
		}
	}

	return nil
}
