// Package xconsensus embeds the x-consensus CosmWasm contract, creates it,
// and copies its params into the x/consensus store once per block.
//
// x-consensus owns Nibiru BFT's consensus params, the role the x/consensus
// module used to have. It holds the block, evidence and validator params
// plus min_block_commit_duration_ms, a chain-wide commit wait that replaces
// each node's config.toml timeout_commit. The x/consensus store remains only
// as the copy that baseapp and Nibiru BFT read, and its MsgUpdateParams is
// blocked by a circuit breaker (app/xconsensus.go).
//
// Only the current x/sudo root may change the contract. It reads the root on
// every execute through the Stargate query /nibiru.sudo.v1.Query/QuerySudoers
// and stores no owner, so a root rotation applies at once. The contract is
// also its own Wasm admin, so the root migrates it through the contract
// (ExecuteMsg.Migrate) and migration rights follow the live root too.
// Governance stays the break-glass for migration.
//
// The module EndBlocker (AppModule, ordered last) queries the contract once
// per block under QueryGasLimit, validates the result with Nibiru BFT's own
// rules, and writes it to the x/consensus store. baseapp reads that store at
// BeginBlock, for every tx, and at EndBlock, where it returns the params to
// Nibiru BFT. So a change executed in block H applies from H+1 for both the
// app and the engine, and a failed query keeps the last good copy. The app
// version is not in the contract: x/upgrade owns it, and the refresh copies
// it unchanged. Contract source: wasm-contracts/x-consensus.
//
// JSON conventions (see wasm-contracts/x-consensus/src/msg.rs):
//   - int64 fields are JSON strings, hence the ",string" tag option, so
//     values above 2^53 survive float-based decoders.
//   - The evidence max age is in nanoseconds, the unit of time.Duration, so
//     it converts without loss.
//   - min_block_commit_duration_ms is a JSON number or null (unset).
//   - The contract rejects unknown fields, so these types must match its
//     messages exactly; the shared fixtures in
//     wasm-contracts/x-consensus/testdata keep them in sync.
package xconsensus

import (
	_ "embed"
	"fmt"
	"time"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
)

// XConsensusWasm is the optimized x-consensus contract bytecode.
//
//   - Source: wasm-contracts/x-consensus (crate x-consensus 0.1.0) at
//     commit fdc94269479510ce55912b442cf95a87b6d873cd.
//   - Build: just rs optimizer build --manifest-path Cargo.toml
//     -p x-consensus --output-dir wasm-contracts/artifacts --mode dev
//     (sai-wasm-optimizer 0.1.0, image cosmwasm/optimizer-arm64:0.17.0,
//     linux/arm64, Rust 1.86.0, Binaryen 116).
//   - SHA-256: 37dc541f06b9229371d7b985b05ecbe76f47be2693dfdfcd0cb8a50196c2096d
//   - Size: 178224 bytes.
//
//go:embed x_consensus.wasm
var XConsensusWasm []byte

// XConsensusWasmSHA256 is the hex SHA-256 of XConsensusWasm. Wasm code
// checksums are SHA-256 of the bytecode, so this is also the Wasm code
// checksum, and it fixes the contract address (PredictContractAddr). The app
// finds the contract through the Wasm creator index (Keeper.ContractAddr),
// not that address, so rebuilding the embedded Wasm is safe; only published
// addresses go stale. A test keeps the constant in sync.
const XConsensusWasmSHA256 = "37dc541f06b9229371d7b985b05ecbe76f47be2693dfdfcd0cb8a50196c2096d"

// MaxMinBlockCommitDurationMs is the contract's upper bound for
// ConsensusParams.MinBlockCommitDurationMs, 30 s. A longer wait would make
// the chain look halted. The Go side checks it again (ValidatedProto), so a
// contract that stopped enforcing it still cannot stretch block times.
const MaxMinBlockCommitDurationMs uint64 = 30_000

// InstantiateMsg instantiates x-consensus. Only EnsureXConsensus sends it,
// seeded from the x/consensus store with the commit wait unset, so creating
// the contract changes no behavior. The contract validates Params exactly
// like an update. Instantiation needs no x/sudo root, so genesis can create
// the contract before a root is set, and the sender gains no rights.
type InstantiateMsg struct {
	// Params are the initial consensus params.
	Params ConsensusParams `json:"params"`
}

// ExecuteMsg is the externally tagged execute message: set exactly one field.
// Only the current x/sudo root may execute it; an empty root or a failed root
// query rejects it. The root check runs before any validation.
type ExecuteMsg struct {
	// UpdateParams replaces all params at once.
	UpdateParams *UpdateParamsMsg `json:"update_params,omitempty"`
	// Migrate migrates x-consensus to new code through the contract itself.
	Migrate *MigrateExecMsg `json:"migrate,omitempty"`
	// UpdateAdmin hands the Wasm admin of x-consensus to another address.
	UpdateAdmin *UpdateAdminMsg `json:"update_admin,omitempty"`
}

// MigrateExecMsg is the body of ExecuteMsg.Migrate, encoded as
// {"migrate":{"code_id":42,"msg":"e30="}}. The contract answers it with a
// Wasm migrate of its own address, which the Wasm keeper accepts while the
// contract is its own admin.
//
// The Wasm keeper treats the contract as the caller, so the new code's
// instantiate permission must allow the x-consensus address, and on mainnet
// the Wasm deployer guard requires the contract to hold the x/sudo
// wasm_deployer role. The new code takes over every rule, including the root
// check.
type MigrateExecMsg struct {
	// CodeID is the code to migrate to.
	CodeID uint64 `json:"code_id"`
	// Msg is the JSON message for the new code's migrate entry point. It
	// encodes as base64, the JSON form of CosmWasm Binary.
	Msg []byte `json:"msg"`
}

// UpdateAdminMsg is the body of ExecuteMsg.UpdateAdmin, encoded as
// {"update_admin":{"admin":"nibi1..."}}. The contract validates the address
// and answers with a Wasm admin change for its own address. Afterwards
// ExecuteMsg.Migrate fails and only the new admin or governance can migrate;
// param updates are unaffected.
type UpdateAdminMsg struct {
	// Admin is the bech32 address of the new Wasm admin.
	Admin string `json:"admin"`
}

// UpdateParamsMsg is the body of ExecuteMsg.UpdateParams. There is no
// partial update: some rules span fields, so the stored params always passed
// validation as a whole. To change one field, send the current params with
// that field changed. The change applies from the next block.
type UpdateParamsMsg struct {
	// Params replace the stored params entirely.
	Params ConsensusParams `json:"params"`
}

// QueryMsg is the externally tagged query message. Use QueryParams(). The
// EndBlocker sends it once per block.
type QueryMsg struct {
	// Params returns ConsensusParams.
	Params *struct{} `json:"params,omitempty"`
}

// QueryParams returns the query for the stored params, encoded as
// {"params":{}}.
func QueryParams() QueryMsg { return QueryMsg{Params: &struct{}{}} }

// MigrateMsg is the message for this code's migrate entry point, encoded as
// {}. It has no fields: migrating into this code keeps the stored params.
type MigrateMsg struct{}

// ConsensusParams is the params object stored in x-consensus and returned
// by the params query. The three groups mirror the block, evidence and
// validator fields of tendermint.types.ConsensusParams (see ToProto), so they
// reach Nibiru BFT through the existing ABCI ConsensusParamUpdates field.
// There is no Version: Nibiru BFT copies Version.App into block headers and
// x/upgrade owns it, so the contract has no power over it. The field docs
// below are short; wasm-contracts/x-consensus/src/msg.rs explains each param
// with citations.
type ConsensusParams struct {
	// Block holds block size and gas limits.
	Block BlockParams `json:"block"`
	// Evidence holds the limits on evidence of validator misbehavior.
	Evidence EvidenceParams `json:"evidence"`
	// Validator holds the public key types validators may use.
	Validator ValidatorParams `json:"validator"`
	// MinBlockCommitDurationMs is the minimum time, in milliseconds, from
	// the commit decision to the start of the next height. Nil (JSON null)
	// means unset: every node keeps its own config.toml timeout_commit.
	//
	// The commit decision is the moment a node first has precommits for a
	// block from more than two thirds of voting power. Executing and saving
	// the block happen inside this window; if they take longer, the next
	// height starts as soon as they finish.
	//
	//	CommitTime = this node first has precommits from >2/3 of voting power
	//	next height scheduled at CommitTime + min_block_commit_duration_ms
	//	  |-- Commit: execute and save the block --|-- NewHeight: wait --|
	//	if execution takes longer, the next height starts as soon as it finishes
	//
	// When set, it replaces config.toml timeout_commit on every node. It is
	// a floor, not an exact length: node-local skip_timeout_commit (once
	// every validator's precommit has arrived) and the catch-up rule
	// (precommits from more than two thirds of voting power for the next
	// height) can still start the next height earlier. Range: 0 to
	// MaxMinBlockCommitDurationMs.
	MinBlockCommitDurationMs *uint64 `json:"min_block_commit_duration_ms"`
}

// BlockParams mirrors tendermint.types.BlockParams.
type BlockParams struct {
	// MaxBytes is the largest block a proposer may build, in bytes, counting
	// header, last commit, evidence and txs. In (0, 100 MiB]. Also the upper
	// bound for EvidenceParams.MaxBytes.
	MaxBytes int64 `json:"max_bytes,string"`
	// MaxGas is the total gas the txs in one block may request; -1 means no
	// limit. The mempool, proposal building, the ante handler and baseapp's
	// block gas meter all use it.
	MaxGas int64 `json:"max_gas,string"`
}

// EvidenceParams mirrors tendermint.types.EvidenceParams. Evidence is too
// old only when it exceeds both age limits; within either, Nibiru BFT accepts
// it and x/evidence slashes for it.
type EvidenceParams struct {
	// MaxAgeNumBlocks is the maximum evidence age in blocks, positive. With
	// block pruning on, baseapp also keeps at least this many recent blocks.
	MaxAgeNumBlocks int64 `json:"max_age_num_blocks,string"`
	// MaxAgeDurationNs is the maximum evidence age in nanoseconds, positive.
	// It is a time.Duration count, kept as int64 to match the JSON field.
	MaxAgeDurationNs int64 `json:"max_age_duration_ns,string"`
	// MaxBytes is the most evidence bytes one block may carry, in
	// [0, Block.MaxBytes]. It is a cap, not a reservation.
	MaxBytes int64 `json:"max_bytes,string"`
}

// ValidatorParams mirrors tendermint.types.ValidatorParams.
type ValidatorParams struct {
	// PubKeyTypes lists the ABCI key type names validators may use
	// ("ed25519", "secp256k1"). It must be non-empty; a nil slice encodes as
	// null, which the contract rejects. Removing a type a bonded validator
	// uses halts the chain at that validator's next power change, since
	// Nibiru BFT rejects the validator update.
	PubKeyTypes []string `json:"pub_key_types"`
}

// MinBlockCommitDuration returns the commit timing as a time.Duration and
// true, or zero and false when unset.
func (p ConsensusParams) MinBlockCommitDuration() (time.Duration, bool) {
	if p.MinBlockCommitDurationMs == nil {
		return 0, false
	}
	return time.Duration(*p.MinBlockCommitDurationMs) * time.Millisecond, true
}

// ToProto converts the params to the engine's proto type with the given
// version, which the caller copies unchanged from the existing store (nil
// leaves it unset). The commit timing has no proto field and is dropped.
func (p ConsensusParams) ToProto(
	version *cmtproto.VersionParams,
) cmtproto.ConsensusParams {
	return cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: p.Block.MaxBytes,
			MaxGas:   p.Block.MaxGas,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: p.Evidence.MaxAgeNumBlocks,
			MaxAgeDuration:  time.Duration(p.Evidence.MaxAgeDurationNs),
			MaxBytes:        p.Evidence.MaxBytes,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: append([]string{}, p.Validator.PubKeyTypes...),
		},
		Version: version,
	}
}

// ConsensusParamsFromProto converts engine params to the contract type with
// the commit timing unset, which is how the upgrade handler and genesis seed
// the contract. Block, Evidence and Validator are required. Version is
// ignored because the contract does not hold it.
func ConsensusParamsFromProto(
	pb cmtproto.ConsensusParams,
) (ConsensusParams, error) {
	if pb.Block == nil || pb.Evidence == nil || pb.Validator == nil {
		return ConsensusParams{}, fmt.Errorf(
			"consensus params missing block, evidence or validator: %v", pb)
	}
	return ConsensusParams{
		Block: BlockParams{
			MaxBytes: pb.Block.MaxBytes,
			MaxGas:   pb.Block.MaxGas,
		},
		Evidence: EvidenceParams{
			MaxAgeNumBlocks:  pb.Evidence.MaxAgeNumBlocks,
			MaxAgeDurationNs: int64(pb.Evidence.MaxAgeDuration),
			MaxBytes:         pb.Evidence.MaxBytes,
		},
		Validator: ValidatorParams{
			PubKeyTypes: append([]string{}, pb.Validator.PubKeyTypes...),
		},
	}, nil
}
