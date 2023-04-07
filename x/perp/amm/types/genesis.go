package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Vpools: []Vpool{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// validate vpools
	vpools := make(map[string]struct{}, len(gs.Vpools))
	for _, p := range gs.Vpools {
		if err := p.Validate(); err != nil {
			return err
		}
		pair := p.Pair.String()
		if _, exists := vpools[pair]; exists {
			return fmt.Errorf("duplicate vpool: %s", pair)
		}
		vpools[pair] = struct{}{}
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
