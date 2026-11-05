use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Int64, Uint64};

/// Scheduling authority and runtime policy for this Chain contract. The native
/// governance module authorizes requests; the live x/sudo root manages policy.
#[cw_serde]
pub struct Config {
    /// Native governance module account allowed to schedule or cancel upgrades.
    pub gov_authority: String,
    /// Warning interval added to the first block reaching a time target.
    /// The contract samples this value once, when selecting the execution height.
    pub activation_delay_blocks: Uint64,
}

/// A concrete software-upgrade plan: the binary name, execution block, and
/// operator metadata consumed by the native upgrade module and node tooling.
/// An unresolved time request is not a Plan. Its target must first resolve to
/// an execution block so validators switch binaries at the same boundary.
#[cw_serde]
pub struct Plan {
    /// Upgrade-handler name and binary-directory identifier used by node tooling.
    pub name: String,
    /// Positive consensus block height at which the native handler must run.
    pub block_height: Int64,
    /// Opaque operator metadata, commonly JSON containing binary download URLs.
    pub info: String,
}

/// Governance-authorized scheduling input. Specify either a positive
/// block_height with no time target, or zero block_height with a future target.
/// Integer wrappers encode JSON numbers as strings to preserve full precision.
#[cw_serde]
pub struct ScheduleRequest {
    /// Name of the native upgrade handler to execute after the binary switch.
    pub name: String,
    /// Requested execution block, or zero when selecting execution by time.
    pub block_height: Int64,
    /// Absolute Unix nanoseconds, compared with consensus block time. This is
    /// not a local-clock deadline; selection adds the configured warning blocks.
    pub target_time_unix_nanos: Option<Uint64>,
    /// Operator metadata copied unchanged into the eventual concrete plan.
    pub info: String,
}

/// The single pending request, including its identity and selection progress.
/// A valid replacement gets another ID so native failure suppression for an
/// earlier request cannot accidentally suppress the replacement.
#[cw_serde]
pub struct PendingSchedule {
    /// Monotonically assigned request identity, unrelated to a proposal number.
    pub id: Uint64,
    /// Name of the native upgrade handler and corresponding binary directory.
    pub name: String,
    /// Fixed execution block, absent until a time request selects its block.
    pub block_height: Option<Int64>,
    /// Original absolute time target in Unix nanoseconds, retained after selection.
    pub target_time_unix_nanos: Option<Uint64>,
    /// Uninterpreted metadata for validators' binary-switch tooling.
    pub info: String,
}

/// An unresolved time request exposed to the native BeginBlock hook. Go compares
/// its target with block time before attempting bounded execution-height selection.
#[cw_serde]
pub struct TimeTarget {
    /// Request identity used to match the native terminal-failure record.
    pub id: Uint64,
    /// Handler name included in selection events and failure diagnostics.
    pub name: String,
    /// Absolute Unix nanoseconds at or after which selection becomes eligible.
    pub target_time_unix_nanos: Uint64,
}

/// An upgrade that completed, also used as the exclusive history-page cursor.
/// Legacy imports can contain the same name at multiple block heights, so the
/// history key includes both values rather than assuming names are unique.
#[cw_serde]
pub struct CompletedUpgrade {
    /// Name of the executed native upgrade handler.
    pub name: String,
    /// Consensus block height at which the upgrade completed.
    pub block_height: Int64,
}

/// One native SDK module's schema version, used by RunMigrations to choose
/// migrations. This is independent of both the chain's software release and
/// the protocol version reported through ABCI.
#[cw_serde]
pub struct ModuleVersion {
    /// Registered native module name.
    pub name: String,
    /// Module consensus/schema version, encoded as a decimal string in JSON.
    pub version: Uint64,
}

/// Logical state for privileged legacy import and verification. Import must
/// preserve an executing plan, so it does not apply ordinary future-time
/// scheduling rules. Wasm genesis export preserves the full raw contract state,
/// including configuration, request IDs, and unresolved targets not in this view.
#[cw_serde]
pub struct Snapshot {
    /// Concrete pending plan, if execution already has a selected block.
    pub plan: Option<Plan>,
    /// Completion records, retaining legacy duplicate-name history.
    pub completed: Vec<CompletedUpgrade>,
    /// Native module schema versions used by migration dispatch.
    pub module_versions: Vec<ModuleVersion>,
    /// ABCI application protocol version. Absence is distinct from explicit zero.
    pub protocol_version: Option<Uint64>,
}

/// Bootstrap input supplied by the native host after module migrations or
/// fresh-chain genesis. An imported existing contract does not run instantiate.
#[cw_serde]
pub struct InstantiateMsg {
    /// Native governance module account trusted for scheduling and cancellation.
    pub gov_authority: String,
    /// Legacy or fresh-genesis state to import without ordinary scheduling checks.
    pub state: Snapshot,
}

/// Externally callable operations. Schedule and Cancel require the governance
/// module account; the other operations query and require the live x/sudo root.
#[cw_serde]
pub enum ExecuteMsg {
    /// Validate and replace the single pending upgrade request atomically.
    Schedule { plan: ScheduleRequest },
    /// Remove the pending request without changing completion history.
    Cancel {},
    /// Set the warning interval for time requests that have not selected a block.
    UpdateConfig { activation_delay_blocks: Uint64 },
    /// Emit a Wasm migration of this contract, which is its own Wasm admin.
    Migrate { code_id: u64, msg: Binary },
    /// Transfer Wasm-admin authority through the root-controlled contract policy.
    UpdateAdmin { admin: String },
}

/// Trusted native lifecycle transitions. The Wasm host, rather than a transaction
/// sender, invokes this entry point. Root-controlled native facilities are part
/// of that trust boundary; these calls do not impersonate governance execute.
#[cw_serde]
pub enum SudoMsg {
    /// Fix execution at current block plus delay when the target time is reached.
    SelectExecutionHeight {},
    /// Merge native migration results without deleting omitted module versions.
    SetModuleVersions { versions: Vec<ModuleVersion> },
    /// Store the protocol version selected by the native upgrade lifecycle.
    SetProtocolVersion { version: Uint64 },
    /// Record completion using the current consensus block height.
    RecordCompleted { name: String },
    /// Clear the pending plan after successful application or skip-height handling.
    ClearPlan {},
}

/// Read-only views for native adapters and operator CLI commands. Rust query
/// responses use the contract schema; Go translates legacy protobuf queries.
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Concrete executable plan, or null while absent or waiting for time selection.
    #[returns(Option<Plan>)]
    CurrentPlan {},
    /// Unresolved time request for the native selection hook, or null.
    #[returns(Option<TimeTarget>)]
    TimeTarget {},
    /// Pending request with its identity, original target, and selected block.
    #[returns(Option<PendingSchedule>)]
    Status {},
    /// Scheduling authority and warning interval.
    #[returns(Config)]
    Config {},
    /// Logical migration-verification view, not a replacement for Wasm genesis.
    #[returns(Snapshot)]
    Snapshot {},
    /// Earliest matching completion height, or zero if absent, preserving SDK behavior.
    #[returns(Int64)]
    Applied { name: String },
    /// Last completion by block height and then name, or null if absent.
    #[returns(Option<CompletedUpgrade>)]
    LastCompleted {},
    /// Completion history ordered by (block_height, name). The cursor is exclusive;
    /// the default page has 50 records and the maximum page has 200 records.
    #[returns(Vec<CompletedUpgrade>)]
    History {
        start_after: Option<CompletedUpgrade>,
        limit: Option<u32>,
        descending: Option<bool>,
    },
    /// All module schema versions in name order, or one named module if present.
    #[returns(Vec<ModuleVersion>)]
    ModuleVersions { name: Option<String> },
    /// Stored ABCI protocol version, retaining the absent-versus-zero distinction.
    #[returns(Option<Uint64>)]
    ProtocolVersion {},
}

/// No-op migration input for this initial contract version. Future contract
/// upgrades must define and test any storage transformation they introduce.
#[cw_serde]
pub struct MigrateMsg {}
