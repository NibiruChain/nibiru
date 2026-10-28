//! contract.rs: Entry points of the x-consensus contract.

use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult,
};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{
    ConsensusParams, ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg,
};
use crate::state::PARAMS;
use crate::sudo::query_sudo_root;

/// cw2 contract name, so tooling can identify the contract on chain.
pub const CONTRACT_NAME: &str = "nibiru-wasm/contracts/x-consensus";
/// cw2 contract version, taken from the crate version.
pub const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Stores the initial params after validating them.
///
/// The instantiating sender gains no rights. Instantiation does not need an
/// x/sudo root, so a fresh chain can create the contract at genesis before
/// any root is set.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    msg.params.validate()?;
    PARAMS.save(deps.storage, &msg.params)?;
    Ok(Response::new().add_attribute("action", "instantiate"))
}

/// Dispatches root-only state changes. The root check runs first, so an
/// unauthorized sender learns nothing about param validity.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    assert_sudo_root(deps.as_ref(), &info)?;
    match msg {
        ExecuteMsg::UpdateParams { params } => update_params(deps, params),
    }
}

/// Fails unless the sender is the current x/sudo root.
///
/// The root is read on every call instead of stored, so a root rotation
/// through x/sudo `MsgChangeRoot` takes effect here at once. An empty root
/// or a failed query rejects the call, so the contract never falls back to
/// letting anyone write.
fn assert_sudo_root(
    deps: Deps,
    info: &MessageInfo,
) -> Result<(), ContractError> {
    let root = query_sudo_root(&deps.querier)?;
    if root.is_empty() {
        return Err(ContractError::NoSudoRoot);
    }
    if info.sender.as_str() != root {
        return Err(ContractError::Unauthorized {
            sender: info.sender.to_string(),
            root,
        });
    }
    Ok(())
}

/// Validates and stores `params`, replacing the stored params entirely.
fn update_params(
    deps: DepsMut,
    params: ConsensusParams,
) -> Result<Response, ContractError> {
    params.validate()?;
    PARAMS.save(deps.storage, &params)?;
    Ok(Response::new().add_attribute("action", "update_params"))
}

/// Answers [`QueryMsg`] from storage. Queries never fail on a live contract,
/// since instantiate always writes the params.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Params {} => to_json_binary(&PARAMS.load(deps.storage)?),
    }
}

/// Records the new cw2 version. Params are left unchanged; a future version
/// that changes the storage layout adds its own migration.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn migrate(
    deps: DepsMut,
    _env: Env,
    _msg: MigrateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    Ok(Response::new().add_attribute("action", "migrate"))
}
