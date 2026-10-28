//! contract.rs: Entry points of the x-consensus contract.

use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response,
    StdResult, WasmMsg,
};

use crate::error::ContractError;
use crate::msg::{
    ConsensusParams, ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg,
};
use crate::state::PARAMS;
use crate::sudo::query_sudo_root;

/// Validates and stores the initial params.
///
/// No root check runs here, and the sender gains no rights. Only the
/// x/xconsensus module creates the contract (at genesis or in the v2.22
/// upgrade), and a new chain may do so before x/sudo has a root. Rights come
/// from the live root at execute time instead.
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

/// Dispatches root-only state changes.
///
/// The root check runs before anything else, including param and address
/// validation, so a sender who is not the root gets the same error for every
/// message and learns nothing about which values would pass.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    assert_sudo_root(deps.as_ref(), &info)?;
    match msg {
        ExecuteMsg::UpdateParams { params } => update_params(deps, params),
        ExecuteMsg::Migrate { code_id, msg } => {
            Ok(migrate_self(&env, code_id, msg))
        }
        ExecuteMsg::UpdateAdmin { admin } => update_admin(deps, &env, admin),
    }
}

/// Fails unless the sender is the current x/sudo root.
///
/// The root is read on every call instead of stored, so a root rotation
/// through x/sudo `MsgChangeRoot` takes effect here at once. An empty root or
/// a failed query rejects the call: the contract never falls back to letting
/// anyone write.
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

/// Validates `params` and stores them in place of the old params.
///
/// Validation runs on the whole set, so a rejected update leaves the old
/// params untouched. The chain picks the new values up at this block's
/// EndBlock and applies them from the next block.
fn update_params(
    deps: DepsMut,
    params: ConsensusParams,
) -> Result<Response, ContractError> {
    params.validate()?;
    PARAMS.save(deps.storage, &params)?;
    Ok(Response::new().add_attribute("action", "update_params"))
}

/// Returns a response that migrates this contract to `code_id` with `msg`.
///
/// The migrate runs as a submessage after this execute returns, with this
/// contract as the sender. The Wasm keeper accepts it only while the
/// contract is its own admin. If the migrate fails, the whole execute fails
/// and nothing changes.
fn migrate_self(env: &Env, code_id: u64, msg: Binary) -> Response {
    Response::new()
        .add_message(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: code_id,
            msg,
        })
        .add_attribute("action", "migrate")
        .add_attribute("new_code_id", code_id.to_string())
}

/// Returns a response that hands the Wasm admin of this contract to
/// `admin`, after checking that `admin` is a valid address on this chain.
///
/// The check rejects a malformed or non-normalized address in the contract,
/// with an error that names it, before any message is dispatched. It cannot
/// catch a valid address that nobody controls, so callers should double
/// check the value: once the admin leaves the contract, only that address or
/// governance can migrate.
fn update_admin(
    deps: DepsMut,
    env: &Env,
    admin: String,
) -> Result<Response, ContractError> {
    let admin = deps.api.addr_validate(&admin)?;
    Ok(Response::new()
        .add_message(WasmMsg::UpdateAdmin {
            contract_addr: env.contract.address.to_string(),
            admin: admin.to_string(),
        })
        .add_attribute("action", "update_admin")
        .add_attribute("admin", admin))
}

/// Answers [`QueryMsg`] from storage.
///
/// The params query cannot fail on a live contract, since instantiate always
/// writes the params. The x/xconsensus EndBlocker sends it once per block
/// under a fixed gas limit, so it stays a single storage read.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Params {} => to_json_binary(&PARAMS.load(deps.storage)?),
    }
}

/// Accepts a migration to this code and keeps the stored params.
///
/// The entry point must exist for the Wasm keeper to migrate into this code.
/// A later version that changes the storage layout adds its state migration
/// here. There is no cw2 version record to check or bump: nothing reads one,
/// and the code checksum already identifies the deployed version.
#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn migrate(
    _deps: DepsMut,
    _env: Env,
    _msg: MigrateMsg,
) -> Result<Response, ContractError> {
    Ok(Response::new().add_attribute("action", "migrate"))
}
