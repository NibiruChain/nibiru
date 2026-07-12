package types

import (
	"time"

	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
)

// NewGenesisState creates an 08-wasm GenesisState instance.
func NewGenesisState(contracts []Contract) *GenesisState {
	return &GenesisState{Contracts: contracts}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for _, contract := range gs.Contracts {
		if err := ValidateWasmCode(contract.CodeBytes); err != nil {
			return sdkioerrors.Wrap(err, "wasm bytecode validation failed")
		}
	}

	return nil
}

// ExportMetadata exports all the consensus metadata in the client store so they
// can be included in clients genesis and imported by a ClientKeeper
func (cs ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	payload := QueryMsg{
		ExportMetadata: &ExportMetadataMsg{},
	}

	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: time.Now()}, true, nil) // context with infinite gas meter
	result, err := wasmQuery[ExportMetadataResult](ctx, store, &cs, payload)
	if err != nil {
		panic(err)
	}

	genesisMetadata := make([]exported.GenesisMetadata, len(result.GenesisMetadata))
	for i, metadata := range result.GenesisMetadata {
		genesisMetadata[i] = metadata
	}

	return genesisMetadata
}
