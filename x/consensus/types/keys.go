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

// ContractLabel is the Wasm label of the x-consensus contract.
const ContractLabel = "x-consensus"

// ContractCreatorName is the name whose module address
// (authtypes.NewModuleAddress) creates x-consensus. It is "xconsensus", not
// ModuleName, on purpose: the creator and ContractSalt fix the contract
// address (see keeper.PredictContractAddr), and the published address
// nibi1mlr9daxryfedvt6sx5g06f22lvtr76x2urdnv5sprs2nyupt86wslq8p0v with
// creator nibi1y0jxhu9gx4hqsx56apkkllfjpctf4ffsk25atd was derived from the
// module's earlier name. Renaming the module must not move the contract, so
// this string is kept separately and a test pins both addresses.
const ContractCreatorName = "xconsensus"

// ContractSalt is the fixed instantiate2 salt. With the fixed creator it
// makes the address depend only on the code checksum, so the address can be
// published before the contract exists. The app itself finds the contract
// through the Wasm creator index (keeper.Keeper.ContractAddr), so a rebuild
// of the embedded Wasm, which changes the address, breaks nothing. It is a
// string constant so no caller can change it; convert with
// []byte(ContractSalt) at use.
const ContractSalt = "x-consensus"

// QueryGasLimit bounds the gas of the per-block params query, in SDK gas.
//
// Measured October 28, 2026 (TestQueryGasUsage): one params query on the
// embedded contract, unpinned, uses 66,013 gas and about 0.1 ms. Most of it
// is the fixed instance setup cost (InstantiateContractCosts); the contract
// itself does one storage read and a JSON encode. 1,000,000 gives 15 times
// headroom for a future contract version with more fields.
//
// The limit is also the worst case a broken contract can cost every block:
// a query that loops runs until the limit, which took about 0.12 s at
// 1,000,000 gas and 0.43 s at 3,000,000 (TestQuerySmartBounded, hackatom
// hashing loop). At the sub-second block times x-consensus exists for, a
// tighter limit keeps a bad migration from multiplying the block time while
// governance fixes it.
//
// It is a Go constant, not a param or node config, so every node meters the
// query the same way: a node-local limit could let some nodes refresh and
// others not, which would split the app hash.
const QueryGasLimit uint64 = 1_000_000

// MaxMinBlockCommitDurationMs is the upper bound for
// ConsensusParams.MinBlockCommitDurationMs, 30 s. A longer wait would make
// the chain look halted.
//
// The contract enforces the same bound (MAX_MIN_BLOCK_COMMIT_DURATION_MS in
// wasm-contracts/x-consensus/src/msg.rs), and the Go side checks it
// again (ConsensusParams.ValidatedProto), so a contract that stopped
// enforcing it still cannot stretch block times. The two copies must match;
// TestContractOnChain checks that the contract rejects one more than this
// value on instantiate and on update.
const MaxMinBlockCommitDurationMs uint64 = 30_000
