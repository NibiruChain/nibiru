// Package types holds the x/consensus messages, queries and store keys.
//
// The proto package stays cosmos.consensus.v1, so type URLs
// (/cosmos.consensus.v1.MsgUpdateParams), the gRPC route
// /cosmos.consensus.v1.Query/Params and the amino name are unchanged. Only
// the Go package moved here from lib/cosmos-sdk/x/consensus/types: Nibiru no
// longer tracks Cosmos SDK compatibility for this module, and the store it
// describes is owned by the Nibiru x/consensus module.
//
// The package must stay light. lib/cosmos-sdk imports it (baseapp tests, the
// deprecated SDK consensus keeper and the depinject test app configs), so it
// must not import x/wasm or anything else that would pull app code into lib.
package types

const (
	// ModuleName defines the name of the x/consensus module.
	ModuleName = "consensus"

	// StoreKey defines the module's store key.
	StoreKey = ModuleName
)

// ParamStoreKeyConsensusParams is the key of the consensus params inside the
// StoreKey store. baseapp reads it at BeginBlock, for every tx and at
// EndBlock, so it must never change.
var ParamStoreKeyConsensusParams = []byte("Consensus")
