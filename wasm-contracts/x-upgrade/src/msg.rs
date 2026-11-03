use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Int64, Uint64};

#[cw_serde]
pub struct Config {
    pub governance_authority: String,
    pub activation_delay_blocks: Uint64,
}

#[cw_serde]
pub struct Plan {
    pub name: String,
    pub height: Int64,
    pub info: String,
}

/// Exactly one of a positive height or target_time is supplied. Timestamp
/// values are consensus Unix nanoseconds encoded as strings, without floats.
#[cw_serde]
pub struct ScheduleRequest {
    pub name: String,
    pub height: Int64,
    pub target_time: Option<Uint64>,
    pub info: String,
}

#[cw_serde]
pub struct PendingSchedule {
    pub id: Uint64,
    pub name: String,
    pub height: Option<Int64>,
    pub target_time: Option<Uint64>,
    pub info: String,
}

#[cw_serde]
pub struct TimeTarget {
    pub id: Uint64,
    pub name: String,
    pub target_time: Uint64,
}

#[cw_serde]
pub struct CompletedUpgrade {
    pub name: String,
    pub height: Int64,
}

#[cw_serde]
pub struct ModuleVersion {
    pub name: String,
    pub version: Uint64,
}

/// Logical state, used for privileged initial import and round-trip tests.
/// Import does not schedule a plan or run normal future-time validation.
#[cw_serde]
pub struct Snapshot {
    pub plan: Option<Plan>,
    pub completed: Vec<CompletedUpgrade>,
    pub module_versions: Vec<ModuleVersion>,
    pub protocol_version: Option<Uint64>,
}

#[cw_serde]
pub struct InstantiateMsg {
    pub governance_authority: String,
    pub state: Snapshot,
}

#[cw_serde]
pub enum ExecuteMsg {
    Schedule { plan: ScheduleRequest },
    Cancel {},
    UpdateConfig { activation_delay_blocks: Uint64 },
    Migrate { code_id: u64, msg: Binary },
    UpdateAdmin { admin: String },
}

/// Trusted native transitions. Native host code and root-controlled host
/// facilities are trusted; this entry point makes no sender-identity claim.
#[cw_serde]
pub enum SudoMsg {
    SelectExecutionHeight {},
    SetModuleVersions { versions: Vec<ModuleVersion> },
    SetProtocolVersion { version: Uint64 },
    RecordCompleted { name: String },
    ClearPlan {},
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(Option<Plan>)]
    CurrentPlan {},
    #[returns(Option<TimeTarget>)]
    TimeTarget {},
    #[returns(Option<PendingSchedule>)]
    Status {},
    #[returns(Config)]
    Config {},
    #[returns(Snapshot)]
    Snapshot {},
    #[returns(Int64)]
    Applied { name: String },
    #[returns(Option<CompletedUpgrade>)]
    LastCompleted {},
    #[returns(Vec<CompletedUpgrade>)]
    History {
        start_after: Option<CompletedUpgrade>,
        limit: Option<u32>,
        descending: Option<bool>,
    },
    #[returns(Vec<ModuleVersion>)]
    ModuleVersions { name: Option<String> },
    #[returns(Option<Uint64>)]
    ProtocolVersion {},
}

#[cw_serde]
pub struct MigrateMsg {}
