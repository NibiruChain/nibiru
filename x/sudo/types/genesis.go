package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
)

func (gen *GenesisState) Validate() error {
	if gen.Sudoers.Contracts == nil {
		return fmt.Errorf("nil contract state must be []string")
	} else if err := gen.Sudoers.Validate(); err != nil {
		return err
	}
	return nil
}

func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
