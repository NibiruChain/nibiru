package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		PostedPrices: []PostedPrice{},
	}
}

// NewGenesisState creates a new genesis state for the pricefeed module
func NewGenesisState(p Params, postedPrices []PostedPrice) *GenesisState {
	var oracles []string
	seenOracles := make(map[string]bool)
	for _, postedPrice := range postedPrices {
		oracle := sdk.MustAccAddressFromBech32(postedPrice.Oracle)
		if seenOracles[oracle.String()] {
			continue
		} else {
			oracles = append(oracles, oracle.String())
		}
		seenOracles[oracle.String()] = true
	}
	return &GenesisState{
		Params:         p,
		PostedPrices:   postedPrices,
		GenesisOracles: oracles,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if err := gs.PostedPrices.Validate(); err != nil {
		return err
	}

	var pairs common.AssetPairs = gs.Params.Pairs // needed for Contains method
	for _, postedPrice := range gs.PostedPrices {
		if !pairs.Contains(common.MustNewAssetPair(postedPrice.PairID)) {
			return fmt.Errorf(
				"pair of posted price, %s, which must be in the genesis params",
				postedPrice.PairID)
		}
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
