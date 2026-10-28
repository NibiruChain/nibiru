// Package xconsensus embeds the x-consensus CosmWasm contract and defines Go
// types for its messages.
//
// x-consensus replaces the storage role of the x/consensus module. It holds
// the Nibiru BFT block, evidence and validator params plus the chain-wide
// min_block_commit_duration_ms. Only the current x/sudo root may update
// them; the contract reads the root on every execute through the Stargate
// query /nibiru.sudo.v1.Query/QuerySudoers and stores no owner. The app
// version is not in the contract: it belongs to x/upgrade. The module
// EndBlocker (AppModule, ordered last) queries the contract once per block,
// validates the result, and writes it to the x/consensus store. That store is
// the copy every reader uses: baseapp at BeginBlock, for every tx, and at
// EndBlock, where it returns the params to Nibiru BFT. So a change applies
// from the next block for both the app and the engine, and a failed query
// keeps the last good copy. Contract source: wasm-contracts/x-consensus.
//
// JSON conventions (see wasm-contracts/x-consensus/src/msg.rs):
//   - int64 fields are JSON strings, hence the ",string" tag option, so
//     values above 2^53 survive float-based decoders.
//   - The evidence max age is in nanoseconds, the unit of time.Duration, so
//     it converts without loss.
//   - min_block_commit_duration_ms is a JSON number or null (unset).
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
//     commit 5cc49aea1bd9154687a596596cc190cff1a4aaf4.
//   - Build: just rs optimizer build --manifest-path Cargo.toml
//     -p x-consensus --output-dir wasm-contracts/artifacts --mode dev
//     (sai-wasm-optimizer 0.1.0, image cosmwasm/optimizer-arm64:0.17.0,
//     linux/arm64, Rust 1.86.0, Binaryen 116).
//   - SHA-256: 9b59483c9064d2baa3c2023ecf90dbc0f1f45ab13acfde14a0c96661e50e11df
//   - Size: 171875 bytes.
//
//go:embed x_consensus.wasm
var XConsensusWasm []byte

// XConsensusWasmSHA256 is the hex SHA-256 of XConsensusWasm. Wasm code
// checksums are SHA-256 of the bytecode, so this is also the Wasm code
// checksum. Nothing depends on the contract address it implies: the app finds
// the contract through the Wasm creator index (Keeper.ContractAddr), so
// rebuilding the embedded Wasm is safe. A test keeps it in sync.
const XConsensusWasmSHA256 = "9b59483c9064d2baa3c2023ecf90dbc0f1f45ab13acfde14a0c96661e50e11df"

// MaxMinBlockCommitDurationMs is the contract's upper bound for
// ConsensusParams.MinBlockCommitDurationMs, 30 s. A longer wait would make
// the chain look halted.
const MaxMinBlockCommitDurationMs uint64 = 30_000

// InstantiateMsg instantiates x-consensus. The contract validates Params
// exactly like an update. Instantiation needs no x/sudo root, so genesis can
// create the contract before a root is set, and the instantiating sender
// gains no rights.
type InstantiateMsg struct {
	// Params are the initial consensus params.
	Params ConsensusParams `json:"params"`
}

// ExecuteMsg is the externally tagged execute message. Only the current
// x/sudo root may execute it; an empty root or a failed root query rejects
// it.
type ExecuteMsg struct {
	// UpdateParams replaces all params at once.
	UpdateParams *UpdateParamsMsg `json:"update_params,omitempty"`
}

// UpdateParamsMsg is the body of ExecuteMsg.UpdateParams. There is no
// partial update, so the stored params always passed validation as a whole.
type UpdateParamsMsg struct {
	// Params replace the stored params entirely.
	Params ConsensusParams `json:"params"`
}

// QueryMsg is the externally tagged query message. Use QueryParams().
type QueryMsg struct {
	// Params returns ConsensusParams.
	Params *struct{} `json:"params,omitempty"`
}

// QueryParams returns the query for the stored params, encoded as
// {"params":{}}.
func QueryParams() QueryMsg { return QueryMsg{Params: &struct{}{}} }

// MigrateMsg is the migrate message. It has no fields: a migration replaces
// the code and keeps the stored params.
type MigrateMsg struct{}

// ConsensusParams is the params object stored in x-consensus and returned
// by the params query. The three groups mirror the block, evidence and
// validator fields of tendermint.types.ConsensusParams (see ToProto). There
// is no Version: the engine copies Version.App into block headers, which
// belongs to x/upgrade, so the contract has no power over it.
type ConsensusParams struct {
	// Block holds block size and gas limits.
	Block BlockParams `json:"block"`
	// Evidence holds evidence age and size limits.
	Evidence EvidenceParams `json:"evidence"`
	// Validator holds the allowed validator public key types.
	Validator ValidatorParams `json:"validator"`
	// MinBlockCommitDurationMs is the minimum time, in milliseconds, from
	// the commit decision to the start of the next height. Nil (JSON null)
	// means unset: every node keeps its own config.toml timeout_commit.
	//
	// The commit decision is the moment a node first has precommits from
	// more than two-thirds of voting power. Executing and saving the block
	// happen inside this window; if they take longer, the next height starts
	// as soon as they finish.
	//
	//	CommitTime = this node first has precommits from >2/3 of voting power
	//	next height scheduled at CommitTime + min_block_commit_duration_ms
	//	  |-- Commit: execute and save the block --|-- NewHeight: wait --|
	//	if execution takes longer, the next height starts as soon as it finishes
	//
	// When set, it replaces config.toml timeout_commit on every node. It is
	// a floor, not an exact length: node-local skip_timeout_commit and the
	// catch-up rule (precommits from more than two-thirds of voting power
	// for the next height) can still start the next height earlier. Range: 0
	// to MaxMinBlockCommitDurationMs.
	MinBlockCommitDurationMs *uint64 `json:"min_block_commit_duration_ms"`
}

// BlockParams mirrors tendermint.types.BlockParams.
type BlockParams struct {
	// MaxBytes is the maximum block size in bytes, in (0, 100 MiB].
	MaxBytes int64 `json:"max_bytes,string"`
	// MaxGas is the maximum gas per block; -1 means unlimited.
	MaxGas int64 `json:"max_gas,string"`
}

// EvidenceParams mirrors tendermint.types.EvidenceParams.
type EvidenceParams struct {
	// MaxAgeNumBlocks is the maximum evidence age in blocks, positive.
	MaxAgeNumBlocks int64 `json:"max_age_num_blocks,string"`
	// MaxAgeDurationNs is the maximum evidence age in nanoseconds, positive.
	// It is a time.Duration count, kept as int64 to match the JSON field.
	MaxAgeDurationNs int64 `json:"max_age_duration_ns,string"`
	// MaxBytes is the maximum evidence size per block, in
	// [0, Block.MaxBytes].
	MaxBytes int64 `json:"max_bytes,string"`
}

// ValidatorParams mirrors tendermint.types.ValidatorParams.
type ValidatorParams struct {
	// PubKeyTypes lists ABCI key type names ("ed25519", "secp256k1"). It
	// must be non-empty; a nil slice encodes as null, which the contract
	// rejects.
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
