//! msg.rs: Messages and the consensus params type of the x-consensus contract.
//!
//! JSON conventions, chosen so the Go adapter can decode the params without
//! loss and without float rounding:
//!
//! - Field names are snake_case and match the proto field names in
//!   `bft/proto/tendermint/types/params.proto`, except where a unit suffix is
//!   added (`max_age_duration_ns`).
//! - Every Go `int64` field is an [`Int64`] and every Go `uint64` field is a
//!   [`Uint64`]. Both serialize as JSON strings, for example `"22020096"`,
//!   because 64-bit values can exceed the largest integer a JSON number holds
//!   exactly (2^53 - 1) in JavaScript and other float-based decoders. This is
//!   the CosmWasm convention and matches how the Cosmos SDK encodes int64 in
//!   JSON. Go decodes them with the `,string` struct tag option.
//! - Durations are integers with the unit in the field name. The evidence
//!   max age is in nanoseconds because Go `time.Duration` is an int64 count of
//!   nanoseconds, so nanoseconds round-trip exactly and any other unit would
//!   either lose precision or need a remainder field.
//! - `min_block_commit_duration_ms` is a plain JSON number because its range
//!   (0 to 30000) is far below 2^53, and it is optional: JSON `null` or a
//!   missing field means unset.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Int64, Uint64};

/// Instantiates x-consensus with an owner and the initial params.
///
/// The chain instantiates it from Go, at genesis (seeded from the params
/// that InitChain stored) or in the upgrade handler (copied from x/consensus),
/// so `params` is validated exactly like an update.
#[cw_serde]
pub struct InstantiateMsg {
    /// Bech32 address allowed to update params and transfer ownership. On
    /// every network this is the x/sudo root, which is the Treasury CW3 on
    /// mainnet and testnet.
    pub owner: String,
    /// Initial consensus params. Must pass [`ConsensusParams::validate`].
    pub params: ConsensusParams,
}

/// Owner-only state changes.
#[cw_serde]
pub enum ExecuteMsg {
    /// Replaces all params at once. There is no partial update, so a stored
    /// params value is always one the owner wrote as a whole and that passed
    /// validation as a whole (some rules, such as evidence max bytes against
    /// block max bytes, span several fields). The chain applies the new
    /// values from the block after the one that executes this message.
    UpdateParams {
        /// New params, replacing the stored params entirely.
        params: ConsensusParams,
    },
    /// Transfers ownership in one step. The sender must be the current
    /// owner. One step matches how the x/sudo root rotates and keeps a CW3
    /// proposal self-contained; the new address is checked as valid bech32.
    UpdateOwner {
        /// Bech32 address of the new owner.
        owner: String,
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
    /// Returns the current owner.
    #[returns(OwnerResponse)]
    Owner {},
}

/// Response to [`QueryMsg::Owner`].
#[cw_serde]
pub struct OwnerResponse {
    /// Bech32 address of the current owner.
    pub owner: String,
}

/// Migration message. It carries no fields: a migration only records the new
/// cw2 contract version. Params and owner stay as stored.
#[cw_serde]
pub struct MigrateMsg {}

/// Every consensus parameter of Nibiru BFT plus the chain-wide commit timing.
///
/// The first four groups mirror `tendermint.types.ConsensusParams`
/// (`bft/proto/tendermint/types/params.proto`) field for field, so the Go
/// adapter converts them one to one and returns them to the engine through
/// the existing ABCI `ConsensusParamUpdates` field.
#[cw_serde]
pub struct ConsensusParams {
    /// Block size and gas limits.
    pub block: BlockParams,
    /// Evidence age and size limits.
    pub evidence: EvidenceParams,
    /// Validator public key types.
    pub validator: ValidatorParams,
    /// App protocol version.
    pub version: VersionParams,
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

/// Mirrors `tendermint.types.VersionParams`.
#[cw_serde]
pub struct VersionParams {
    /// App protocol version reported to the engine. `bft/types/params.go`
    /// `Update` copies it into the engine state, so the stored value must be
    /// the real app version.
    pub app: Uint64,
}
