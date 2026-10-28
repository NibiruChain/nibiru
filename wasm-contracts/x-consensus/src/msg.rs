//! msg.rs: Messages and the consensus params type of the x-consensus contract.
//!
//! JSON conventions. The Go module `x/xconsensus` decodes these messages, so
//! each choice keeps values exact across Rust, Go and JavaScript clients:
//!
//! - Field names are snake_case and match the proto field names in
//!   `bft/proto/tendermint/types/params.proto`, except where a unit suffix is
//!   added (`max_age_duration_ns`).
//! - Every Go `int64` field is an [`Int64`], which serializes as a JSON
//!   string, for example `"22020096"`. A JSON number above 2^53 - 1 loses
//!   precision in JavaScript and other float-based decoders, and some of
//!   these fields legitimately reach the int64 range (`max_age_duration_ns`
//!   for long ages). Strings are the CosmWasm convention and match how the
//!   Cosmos SDK encodes int64 in JSON. Go decodes them with the `,string`
//!   struct tag option.
//! - Durations are integers with the unit in the field name. The evidence
//!   max age is in nanoseconds because Go `time.Duration` is an int64 count
//!   of nanoseconds, so the value converts both ways without rounding.
//! - `min_block_commit_duration_ms` is a plain JSON number, because its
//!   range (0 to 30000) is far below 2^53, and it is optional: JSON `null`
//!   or a missing field means unset.
//! - Every message type uses `cw_serde`, which rejects unknown fields. A
//!   typo such as `max_byte` fails loudly instead of being dropped, and a
//!   params object that still carries `version` is rejected rather than
//!   silently losing it.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Int64};

/// Creates x-consensus with its initial params.
///
/// The chain instantiates the contract from Go: at genesis, seeded from the
/// params that InitChain stored in x/consensus, or in the v2.22 upgrade
/// handler, copied from the existing x/consensus store. Either way
/// `min_block_commit_duration_ms` starts unset, so creating the contract
/// changes no behavior.
///
/// There is no owner field and no root check here. The right to change
/// params is checked against the live x/sudo root at execute time, and a new
/// chain may create the contract before any root is set.
#[cw_serde]
pub struct InstantiateMsg {
    /// Initial consensus params. Must pass [`ConsensusParams::validate`],
    /// the same check as an update.
    pub params: ConsensusParams,
}

/// State changes. Only the current x/sudo root may send them.
///
/// The contract stores no owner. It reads the root from x/sudo on every
/// execute, because x/sudo `MsgChangeRoot` rewrites only the x/sudo root, so
/// a stored owner would go stale after a rotation.
#[cw_serde]
pub enum ExecuteMsg {
    /// Replaces all params at once.
    ///
    /// There is no partial update. Some rules span several fields, such as
    /// `evidence.max_bytes` at most `block.max_bytes`, so the stored params
    /// are always a set the root wrote whole and that passed validation
    /// whole. A caller changing one field sends the current params with that
    /// field changed.
    ///
    /// The chain applies the new values from the block after the one that
    /// executes this message, for both the app and Nibiru BFT.
    UpdateParams {
        /// New params, replacing the stored params entirely.
        params: ConsensusParams,
    },

    /// Migrates this contract to `code_id`.
    ///
    /// The contract is its own Wasm admin, so it answers with a Wasm migrate
    /// message for its own address, which the Wasm keeper accepts. This makes
    /// migration follow the live x/sudo root, like param updates, instead of
    /// an admin address fixed at creation.
    ///
    /// The Wasm keeper checks the migration from the contract's point of
    /// view, since the contract sends it:
    ///
    /// - The new code's instantiate permission must allow this contract's
    ///   address. This is an operator step when storing the code, for example
    ///   `nibid tx wasm store new.wasm --instantiate-anyof-addresses
    ///   <x-consensus address>` (or `--instantiate-everybody`).
    /// - On mainnet, the Wasm deployer guard checks the migration's actor,
    ///   which is this contract, not the root. The chain grants the contract
    ///   the x/sudo `wasm_deployer` role when it creates it (and on any later
    ///   `EnsureXConsensus`), so no manual step is needed. The role gives the
    ///   contract no power beyond migrating itself on the root's order,
    ///   because it can only send these two messages for its own address.
    ///
    /// The new code takes over every rule, including this root check, so a
    /// migration must keep `assert_sudo_root` and these variants unless it
    /// means to change who controls the contract.
    Migrate {
        /// Code ID to migrate to.
        code_id: u64,
        /// JSON message for the new code's `migrate` entry point, as
        /// base64 in JSON. For this code it is `{}` ([`MigrateMsg`]).
        msg: Binary,
    },

    /// Hands the Wasm admin of this contract to `admin`.
    ///
    /// After this, the contract is no longer its own admin, so
    /// [`ExecuteMsg::Migrate`] fails and only `admin` (or governance) can
    /// migrate. Param updates are unaffected: they never depend on the Wasm
    /// admin. Handing the admin back to the contract's own address restores
    /// the self-admin model. There is no variant to clear the admin, because
    /// that would leave governance as the only way to migrate.
    UpdateAdmin {
        /// Bech32 address of the new Wasm admin. Validated before dispatch.
        admin: String,
    },
}

/// Read-only queries. The x/xconsensus EndBlocker sends [`QueryMsg::Params`]
/// once per block.
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Returns the stored [`ConsensusParams`].
    #[returns(ConsensusParams)]
    Params {},
}

/// Message for the `migrate` entry point. It has no fields, because a
/// migration to this code keeps the stored params as they are. It still
/// rejects unknown fields, so `{}` is the only valid value.
#[cw_serde]
pub struct MigrateMsg {}

/// Every consensus param of Nibiru BFT that the chain governs, plus the
/// chain-wide commit wait.
///
/// The three groups mirror `tendermint.types.ConsensusParams`
/// (`bft/proto/tendermint/types/params.proto`) field for field, so the Go
/// module converts them one to one and returns them to Nibiru BFT through
/// the existing ABCI `ConsensusParamUpdates` field. No proto changes were
/// needed.
///
/// `version` (the app protocol version) is left out on purpose. Nibiru BFT
/// copies `Version.App` into every block header
/// (`bft/state/execution.go:529`), and x/upgrade owns that value. The Go
/// module copies it unchanged from the x/consensus store, so this contract
/// has no power over it. There is no version field for the contract itself
/// either: the Wasm code checksum identifies the deployed code.
#[cw_serde]
pub struct ConsensusParams {
    /// Block size and gas limits.
    pub block: BlockParams,
    /// Limits on evidence of validator misbehavior.
    pub evidence: EvidenceParams,
    /// Public key types validators may use.
    pub validator: ValidatorParams,
    /// Minimum time, in milliseconds, from the commit decision to the start
    /// of the next height. `None` (JSON `null` or absent) means unset: every
    /// node keeps its own `config.toml` `timeout_commit`.
    ///
    /// The commit decision is the moment a node first has precommits for a
    /// block from more than two thirds of voting power
    /// (`bft/consensus/state.go:1554` sets `CommitTime`). The node schedules
    /// the next height at `CommitTime` plus this value
    /// (`bft/consensus/commit_timeout.go:73`). Executing and saving the block
    /// happen inside that window, so this is not a pause added after the
    /// block is saved:
    ///
    /// ```text
    /// CommitTime = this node first has precommits from >2/3 of voting power
    /// next height scheduled at CommitTime + min_block_commit_duration_ms
    ///   |-- Commit: execute and save the block --|-- NewHeight: wait --|
    /// if execution takes longer, the next height starts as soon as it finishes
    /// ```
    ///
    /// When set, it replaces `config.toml` `timeout_commit` on every node, so
    /// all validators pace blocks the same way. It is a floor, not an exact
    /// length. Two rules can still start the next height earlier:
    /// node-local `skip_timeout_commit`, once precommits from every validator
    /// have arrived (`bft/consensus/state.go:2107`), and the catch-up rule,
    /// when a node sees precommits from more than two thirds of voting power
    /// for the height it is waiting to start (`bft/consensus/state.go:2229`).
    ///
    /// Lower values mean shorter block times but less time for
    /// transactions to reach the next proposer. 0 starts the next height as
    /// soon as the block is saved.
    ///
    /// Bounds: 0 to [`MAX_MIN_BLOCK_COMMIT_DURATION_MS`] (30 s) inclusive. A
    /// longer wait would make the chain look halted, and no use case needs
    /// one.
    pub min_block_commit_duration_ms: Option<u64>,
}

/// Upper bound for [`ConsensusParams::min_block_commit_duration_ms`], 30 s.
pub const MAX_MIN_BLOCK_COMMIT_DURATION_MS: u64 = 30_000;

/// Mirrors `tendermint.types.BlockParams`: how big and how expensive one
/// block may be.
#[cw_serde]
pub struct BlockParams {
    /// Largest block a proposer may build, in bytes, counting the header,
    /// the last commit, evidence and transactions. Must be in (0, 100 MiB]
    /// (`MaxBlockSizeBytes`, `bft/types/params.go:16`).
    ///
    /// The proposer fills the block with transactions up to this size minus
    /// the header, commit and evidence (`bft/state/execution.go:102-110`).
    /// Other nodes reject a proposal larger than it
    /// (`bft/consensus/state.go:1954`), and the mempool rejects a transaction
    /// too large to fit in any block (`bft/state/tx_filter.go:11`). It is also
    /// the upper bound for [`EvidenceParams::max_bytes`].
    ///
    /// Validation accepts tiny values, but a block must at least hold its
    /// header and last commit: a value below that makes Nibiru BFT panic when
    /// it builds a proposal (`bft/types/block.go:291`).
    pub max_bytes: Int64,
    /// Total gas the transactions in one block may request. -1 means no
    /// block gas limit. Must be at least -1.
    ///
    /// The mempool rejects a transaction whose gas wanted exceeds it
    /// (`bft/mempool/mempool.go:140`), proposals stop adding transactions
    /// at it (`app/prepare_proposal.go:108`), the ante handler rejects a
    /// transaction whose gas limit exceeds it
    /// (`lib/cosmos-sdk/x/auth/ante/setup.go:45`), and baseapp meters each
    /// block against it (`lib/cosmos-sdk/baseapp/baseapp.go:576`). 0 is valid
    /// but leaves no room for any transaction that requests gas.
    pub max_gas: Int64,
}

/// Mirrors `tendermint.types.EvidenceParams`: which evidence of validator
/// misbehavior, such as a double sign, is still accepted and slashed.
///
/// Evidence is too old only when it is older than both
/// `max_age_num_blocks` and `max_age_duration_ns`. While it is within either
/// limit, Nibiru BFT accepts it (`bft/evidence/verify.go:40`) and x/evidence
/// slashes for it (`lib/cosmos-sdk/x/evidence/keeper/infraction.go:64`). So
/// the longer of the two windows decides. A short window lets a double sign
/// escape slashing.
#[cw_serde]
pub struct EvidenceParams {
    /// Maximum evidence age in blocks. Must be positive.
    ///
    /// When block pruning is on (`min-retain-blocks` above 0), baseapp also
    /// keeps at least this many recent blocks, so evidence can still be
    /// checked against them (`lib/cosmos-sdk/baseapp/abci.go:900`).
    pub max_age_num_blocks: Int64,
    /// Maximum evidence age in nanoseconds, measured from the misbehavior's
    /// block time to the last block time. Must be positive.
    ///
    /// The proto field is `max_age_duration`, a `google.protobuf.Duration`.
    /// Nanoseconds are the unit of Go `time.Duration`, so the value converts
    /// exactly, and the `_ns` suffix names the unit.
    pub max_age_duration_ns: Int64,
    /// Maximum total size of evidence in one block, in bytes. Must be in
    /// [0, block.max_bytes].
    ///
    /// The proposer includes pending evidence up to this size
    /// (`bft/state/execution.go:105`), and nodes reject a block whose
    /// evidence exceeds it (`bft/state/validation.go:148`). It is a cap, not
    /// a reservation: transactions use whatever space evidence does not.
    pub max_bytes: Int64,
}

/// Mirrors `tendermint.types.ValidatorParams`.
#[cw_serde]
pub struct ValidatorParams {
    /// ABCI public key type names validators may use for consensus. Must be
    /// non-empty, and each entry must be a type Nibiru BFT knows: `ed25519`
    /// or `secp256k1`.
    ///
    /// x/staking rejects `MsgCreateValidator` with a key type not listed
    /// (`lib/cosmos-sdk/x/staking/keeper/msg_server.go:74`). Nibiru BFT also
    /// checks every validator power update against this list and stops the
    /// chain on a mismatch (`bft/state/execution.go:483`,
    /// `bft/consensus/state.go:1723`). So removing a type that a bonded
    /// validator uses, such as `ed25519` today, would halt the chain at that
    /// validator's next power change, not only block new validators.
    pub pub_key_types: Vec<String>,
}
