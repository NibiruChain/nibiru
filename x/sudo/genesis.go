package sudo

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
)

func (gen *GenesisState) Validate() error {
	if gen.Sudoers.Contracts == nil {
		return ErrGenesis("nil contract state must be []string")
	} else if err := gen.Sudoers.Validate(); err != nil {
		return ErrGenesis(err.Error())
	}
	if gen.ZeroGasActors != nil {
		err := gen.ZeroGasActors.Validate()
		if err != nil {
			return ErrGenesis(err.Error())
		}
	}
	return nil
}

// DefaultGenesis: A blank genesis state. The DefaultGenesis is invalid because
// it does not specify a "Sudoers.Root".
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Sudoers: Sudoers{
			Root:      "",
			Contracts: []string{},
		},
	}
}

func GetGenesisStateFromAppState(
	cdc codec.JSONCodec,
	appState map[string]json.RawMessage,
) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
