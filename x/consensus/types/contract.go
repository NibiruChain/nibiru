package types

// This file holds the Go side of the x-consensus contract messages and the
// conversions between its params and Nibiru BFT's proto type. It lives in the
// light types package, not the keeper, so clients and lib code can encode
// contract messages without importing x/wasm.
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

import (
	"fmt"
	"time"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
)

// InstantiateMsg instantiates x-consensus. Only
// keeper.Keeper.EnsureXConsensus sends it, seeded from the x/consensus store
// with the commit wait unset, so creating the contract changes no behavior. The contract validates Params exactly
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
// The Wasm keeper treats the contract as the caller. So the new code's
// instantiate permission must allow the x-consensus address, an operator step
// when storing it (nibid tx wasm store --instantiate-anyof-addresses
// <x-consensus address>). The mainnet Wasm deployer guard also requires the
// caller to hold the x/sudo wasm_deployer role, which EnsureXConsensus grants
// the contract on every chain. The new code takes over every rule, including
// the root check.
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

// ValidatedProto converts the contract params to the engine's proto type with
// the given version (copied from the store by the caller) and checks them the
// way Nibiru BFT will: ValidateBasic on the full params struct, since the
// engine halts on an invalid update, plus MaxMinBlockCommitDurationMs. The
// contract enforces the same rules; checking again here means a contract bug
// or a bad migration cannot halt the chain.
func (p ConsensusParams) ValidatedProto(
	version *cmtproto.VersionParams,
) (cmtproto.ConsensusParams, error) {
	if ms := p.MinBlockCommitDurationMs; ms != nil &&
		*ms > MaxMinBlockCommitDurationMs {
		return cmtproto.ConsensusParams{}, fmt.Errorf(
			"min_block_commit_duration_ms %d exceeds the %d ms cap",
			*ms, MaxMinBlockCommitDurationMs)
	}
	pb := p.ToProto(version)
	// bft/types ConsensusParamsFromProto dereferences Version; validation
	// does not look at it, so a nil version validates as the zero version.
	full := pb
	if full.Version == nil {
		full.Version = &cmtproto.VersionParams{}
	}
	if err := cmttypes.ConsensusParamsFromProto(full).ValidateBasic(); err != nil {
		return cmtproto.ConsensusParams{}, fmt.Errorf(
			"invalid x-consensus params: %w", err)
	}
	return pb, nil
}
