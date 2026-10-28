//! contract.rs: Entry points of the x-consensus contract.

use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult,
};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{
    ConsensusParams, ExecuteMsg, InstantiateMsg, MigrateMsg, OwnerResponse,
    QueryMsg,
};
use crate::state::{OWNER, PARAMS};

/// cw2 contract name, so tooling can identify the contract on chain.
pub const CONTRACT_NAME: &str = "nibiru-wasm/contracts/x-consensus";
/// cw2 contract version, taken from the crate version.
pub const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Stores the owner and the initial params after validating both.
///
/// The instantiating sender has no special rights; only `msg.owner` does.
/// This lets the chain instantiate from Go with any creator while the x/sudo
/// root owns the contract.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    let owner = deps.api.addr_validate(&msg.owner)?;
    msg.params.validate()?;
    OWNER.save(deps.storage, &owner)?;
    PARAMS.save(deps.storage, &msg.params)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", owner))
}

/// Dispatches owner-only state changes. Every branch checks the owner first,
/// so an unauthorized sender learns nothing about param validity.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    assert_owner(deps.as_ref(), &info)?;
    match msg {
        ExecuteMsg::UpdateParams { params } => update_params(deps, params),
        ExecuteMsg::UpdateOwner { owner } => update_owner(deps, owner),
    }
}

/// Returns [`ContractError::Unauthorized`] unless the sender is the owner.
fn assert_owner(deps: Deps, info: &MessageInfo) -> Result<(), ContractError> {
    let owner = OWNER.load(deps.storage)?;
    if info.sender != owner {
        return Err(ContractError::Unauthorized {
            sender: info.sender.to_string(),
            owner: owner.to_string(),
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

/// Validates `owner` as a bech32 address and makes it the owner.
fn update_owner(
    deps: DepsMut,
    owner: String,
) -> Result<Response, ContractError> {
    let owner = deps.api.addr_validate(&owner)?;
    OWNER.save(deps.storage, &owner)?;
    Ok(Response::new()
        .add_attribute("action", "update_owner")
        .add_attribute("owner", owner))
}

/// Answers [`QueryMsg`] from storage. Queries never fail on a live contract,
/// since instantiate always writes both items.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Params {} => to_json_binary(&PARAMS.load(deps.storage)?),
        QueryMsg::Owner {} => to_json_binary(&OwnerResponse {
            owner: OWNER.load(deps.storage)?.to_string(),
        }),
    }
}

/// Records the new cw2 version. Params and owner are left unchanged; a
/// future version that changes the storage layout adds its own migration.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn migrate(
    deps: DepsMut,
    _env: Env,
    _msg: MigrateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    Ok(Response::new().add_attribute("action", "migrate"))
}
