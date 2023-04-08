package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Markets: []Market{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// validate markets
	markets := make(map[string]struct{}, len(gs.Markets))
	for _, p := range gs.Markets {
		if err := p.Validate(); err != nil {
			return err
		}
		pair := p.Pair.String()
		if _, exists := markets[pair]; exists {
			return fmt.Errorf("duplicate vpool: %s", pair)
		}
		markets[pair] = struct{}{}
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
