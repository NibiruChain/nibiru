//! contract.rs: Entry points of the x-consensus contract.

use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult,
};

use crate::error::ContractError;
use crate::msg::{
    ConsensusParams, ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg,
};
use crate::state::PARAMS;
use crate::sudo::query_sudo_root;

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

/// Accepts a code migration and leaves the params unchanged. The entry point
/// exists so the Wasm admin can replace the code; a future version that
/// changes the storage layout adds its own state migration here. There is no
/// cw2 version record: nothing reads one, and the code hash already
/// identifies the deployed version.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn migrate(
    _deps: DepsMut,
    _env: Env,
    _msg: MigrateMsg,
) -> Result<Response, ContractError> {
    Ok(Response::new().add_attribute("action", "migrate"))
}
