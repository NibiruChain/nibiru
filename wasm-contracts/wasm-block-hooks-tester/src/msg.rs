use cosmwasm_schema::{cw_serde, QueryResponses};
use serde_json::Value;

/// InstantiateMsg creates a blank fixture. Tests configure behavior later with
/// ExecuteMsg::Config so every scenario has explicit setup.
#[cw_serde]
pub struct InstantiateMsg {}

/// ExecuteMsg updates fixture settings so tests can switch scenarios without
/// redeploying the contract.
#[cw_serde]
pub enum ExecuteMsg {
    /// Config is the single test setup entrypoint. None leaves a field
    /// unchanged; Some(vec![]) explicitly configures an empty registry response.
    Config {
        count: Option<u64>,
        query_error: Option<bool>,
        wasm_sudo_msg_calls: Option<Vec<WasmSudoMsg>>,
    },
}

/// SudoMsg is the target-contract message schema the Go host passes into the
/// fixture sudo entry point.
#[cw_serde]
pub enum SudoMsg {
    /// Increment adds `by` to the counter and records a successful sudo call.
    Increment { by: u64 },
    /// Set overwrites the counter and records a successful sudo call.
    Set { count: u64 },
    /// FailBeforeWrite returns an error before mutating state.
    FailBeforeWrite {},
    /// FailAfterWrite mutates state and then returns an error, allowing host
    /// tests to prove rollback behavior.
    FailAfterWrite { by: u64 },
}

/// WasmSudoMsg is one registry-selected target sudo call.
#[cw_serde]
pub struct WasmSudoMsg {
    /// Bech32 address of the target Wasm contract.
    pub contract_addr: String,
    /// JSON sudo message payload to pass to the target contract.
    pub msg: Value,
}

/// QueryMsg exposes registry planning queries and state inspection.
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// BeginBlockPlan returns wasm_sudo_msg_calls for the begin-block hook.
    #[returns(Vec<WasmSudoMsg>)]
    BeginBlockPlan {},

    /// EndBlockPlan returns wasm_sudo_msg_calls for the end-block hook.
    #[returns(Vec<WasmSudoMsg>)]
    EndBlockPlan {},

    /// State returns fixture state for tests.
    #[returns(crate::state::State)]
    State {},
}
