//! msg.rs: Messages and the consensus params type of the x-consensus contract.
//!
//! JSON conventions, chosen so the Go adapter can decode the params without
//! loss and without float rounding:
//!
//! - Field names are snake_case and match the proto field names in
//!   `bft/proto/tendermint/types/params.proto`, except where a unit suffix is
//!   added (`max_age_duration_ns`).
//! - Every Go `int64` field is an [`Int64`], which serializes as a JSON
//!   string, for example `"22020096"`, because 64-bit values can exceed the
//!   largest integer a JSON number holds exactly (2^53 - 1) in JavaScript and
//!   other float-based decoders. This is the CosmWasm convention and matches
//!   how the Cosmos SDK encodes int64 in JSON. Go decodes them with the
//!   `,string` struct tag option.
//! - Durations are integers with the unit in the field name. The evidence
//!   max age is in nanoseconds because Go `time.Duration` is an int64 count of
//!   nanoseconds, so nanoseconds round-trip exactly and any other unit would
//!   either lose precision or need a remainder field.
//! - `min_block_commit_duration_ms` is a plain JSON number because its range
//!   (0 to 30000) is far below 2^53, and it is optional: JSON `null` or a
//!   missing field means unset.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Int64;

/// Instantiates x-consensus with the initial params.
///
/// The chain instantiates it from Go, at genesis (seeded from the params
/// that InitChain stored) or in the upgrade handler (copied from x/consensus),
/// so `params` is validated exactly like an update. There is no owner field:
/// the right to update is checked against the x/sudo root at execute time.
#[cw_serde]
pub struct InstantiateMsg {
    /// Initial consensus params. Must pass [`ConsensusParams::validate`].
    pub params: ConsensusParams,
}

/// State changes. Only the current x/sudo root may execute them.
///
/// The contract stores no owner. It reads the root from x/sudo on every
/// execute, because x/sudo `MsgChangeRoot` rewrites only the sudo root, so a
/// stored owner would go stale after a root rotation, and a fresh chain may
/// start with no root at all.
#[cw_serde]
pub enum ExecuteMsg {
    /// Replaces all params at once. There is no partial update, so a stored
    /// params value is always one the root wrote as a whole and that passed
    /// validation as a whole (some rules, such as evidence max bytes against
    /// block max bytes, span several fields). The chain applies the new
    /// values from the block after the one that executes this message.
    UpdateParams {
        /// New params, replacing the stored params entirely.
        params: ConsensusParams,
    },
}

/// Read-only queries. The Go adapter calls [`QueryMsg::Params`] once per
/// block at EndBlock.
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Returns the stored [`ConsensusParams`].
    #[returns(ConsensusParams)]
    Params {},
}

/// Migration message. It carries no fields: a migration replaces the code and
/// keeps the stored params.
#[cw_serde]
pub struct MigrateMsg {}

/// Every consensus parameter of Nibiru BFT plus the chain-wide commit timing.
///
/// The three groups mirror `tendermint.types.ConsensusParams`
/// (`bft/proto/tendermint/types/params.proto`) field for field, so the Go
/// adapter converts them one to one and returns them to the engine through
/// the existing ABCI `ConsensusParamUpdates` field.
///
/// `version` (the app protocol version) is deliberately absent. The engine
/// copies `Version.App` into block headers, and that value belongs to
/// x/upgrade, so the contract has no power over it. The Go adapter copies
/// the version unchanged from the existing store.
#[cw_serde]
pub struct ConsensusParams {
    /// Block size and gas limits.
    pub block: BlockParams,
    /// Evidence age and size limits.
    pub evidence: EvidenceParams,
    /// Validator public key types.
    pub validator: ValidatorParams,
    /// Minimum time, in milliseconds, from the commit decision to the start
    /// of the next height. `None` (JSON `null` or absent) means unset: every
    /// node keeps its own `config.toml` `timeout_commit`.
    ///
    /// The commit decision is the moment a node first has precommits from
    /// more than two-thirds of voting power. Executing and saving the block
    /// happen inside this window; if they take longer, the next height starts
    /// as soon as they finish.
    ///
    /// ```text
    /// CommitTime = this node first has precommits from >2/3 of voting power
    /// next height scheduled at CommitTime + min_block_commit_duration_ms
    ///   |-- Commit: execute and save the block --|-- NewHeight: wait --|
    /// if execution takes longer, the next height starts as soon as it finishes
    /// ```
    ///
    /// When set, it replaces `config.toml` `timeout_commit` on every node. It
    /// is a floor, not an exact length: node-local `skip_timeout_commit`
    /// (once every validator's precommit has arrived) and the catch-up rule
    /// (precommits from more than two-thirds of voting power for the next
    /// height) can still start the next height earlier.
    ///
    /// Bounds: 0 to [`MAX_MIN_BLOCK_COMMIT_DURATION_MS`] (30 s) inclusive. A
    /// longer wait would make the chain look halted.
    pub min_block_commit_duration_ms: Option<u64>,
}

/// Upper bound for [`ConsensusParams::min_block_commit_duration_ms`], 30 s.
pub const MAX_MIN_BLOCK_COMMIT_DURATION_MS: u64 = 30_000;

/// Mirrors `tendermint.types.BlockParams`.
#[cw_serde]
pub struct BlockParams {
    /// Maximum block size in bytes. Must be in (0, 100 MiB]
    /// (`MaxBlockSizeBytes` in `bft/types/params.go`).
    pub max_bytes: Int64,
    /// Maximum gas per block. -1 means unlimited.
    pub max_gas: Int64,
}

/// Mirrors `tendermint.types.EvidenceParams`. Evidence is accepted only when
/// it is younger than both `max_age_num_blocks` and `max_age_duration_ns`.
#[cw_serde]
pub struct EvidenceParams {
    /// Maximum evidence age in blocks. Must be positive.
    pub max_age_num_blocks: Int64,
    /// Maximum evidence age in nanoseconds, the unit of Go `time.Duration`,
    /// so the value converts exactly. Must be positive. The proto field is
    /// `max_age_duration` (a `google.protobuf.Duration`); the `_ns` suffix
    /// names the unit.
    pub max_age_duration_ns: Int64,
    /// Maximum total evidence size per block in bytes. Must be in
    /// [0, block.max_bytes].
    pub max_bytes: Int64,
}

/// Mirrors `tendermint.types.ValidatorParams`.
#[cw_serde]
pub struct ValidatorParams {
    /// ABCI public key type names validators may use. Must be non-empty, and
    /// each entry must be a known type: `ed25519` or `secp256k1`.
    pub pub_key_types: Vec<String>,
}
