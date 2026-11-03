use crate::msg::*;
use crate::state::*;
use crate::sudo::query_sudo_root;
use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, Event, Int64, MessageInfo,
    Order, Response, StdError, StdResult, Uint64, WasmMsg,
};
use cw_storage_plus::Bound;

fn invalid(message: &str) -> StdError {
    StdError::generic_err(message)
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    deps.api.addr_validate(&msg.governance_authority)?;
    CONFIG.save(
        deps.storage,
        &Config {
            governance_authority: msg.governance_authority,
            activation_delay_blocks: Uint64::new(1000),
        },
    )?;
    NEXT_ID.save(deps.storage, &1)?;
    if let Some(plan) = msg.state.plan {
        if plan.name.is_empty() || plan.height.i64() <= 0 {
            return Err(invalid("invalid imported plan"));
        }
        PENDING.save(
            deps.storage,
            &PendingSchedule {
                id: Uint64::new(1),
                name: plan.name,
                height: Some(plan.height),
                target_time: None,
                info: plan.info,
            },
        )?;
    }
    for record in msg.state.completed {
        if record.name.is_empty() || record.height.i64() <= 0 {
            return Err(invalid("invalid imported completion"));
        }
        history().save(
            deps.storage,
            (record.height.i64() as u64, record.name.clone()),
            &record,
        )?;
    }
    set_versions(deps.storage, msg.state.module_versions)?;
    if let Some(version) = msg.state.protocol_version {
        PROTOCOL_VERSION.save(deps.storage, &version)?;
    }
    Ok(Response::new().add_attribute("action", "instantiate"))
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::Schedule { plan } => {
            assert_governance(deps.as_ref(), &info)?;
            schedule(deps, env, plan)
        }
        ExecuteMsg::Cancel {} => {
            assert_governance(deps.as_ref(), &info)?;
            PENDING.remove(deps.storage);
            Ok(Response::new().add_attribute("action", "cancel"))
        }
        other => {
            assert_root(deps.as_ref(), &info)?;
            match other {
                ExecuteMsg::UpdateConfig {
                    activation_delay_blocks,
                } => {
                    if activation_delay_blocks.u64() < 2
                        || activation_delay_blocks.u64() > i64::MAX as u64
                    {
                        return Err(invalid(
                            "activation delay must be between 2 and int64 max",
                        ));
                    }
                    CONFIG.update(deps.storage, |mut c| -> StdResult<_> {
                        c.activation_delay_blocks = activation_delay_blocks;
                        Ok(c)
                    })?;
                    Ok(Response::new().add_attribute("action", "update_config"))
                }
                ExecuteMsg::Migrate { code_id, msg } => Ok(Response::new()
                    .add_message(WasmMsg::Migrate {
                        contract_addr: env.contract.address.to_string(),
                        new_code_id: code_id,
                        msg,
                    })),
                ExecuteMsg::UpdateAdmin { admin } => {
                    let admin = deps.api.addr_validate(&admin)?;
                    Ok(Response::new().add_message(WasmMsg::UpdateAdmin {
                        contract_addr: env.contract.address.to_string(),
                        admin: admin.to_string(),
                    }))
                }
                _ => unreachable!(),
            }
        }
    }
}

fn assert_governance(deps: Deps, info: &MessageInfo) -> StdResult<()> {
    if info.sender.as_str() != CONFIG.load(deps.storage)?.governance_authority {
        return Err(invalid("sender is not governance authority"));
    }
    Ok(())
}
fn assert_root(deps: Deps, info: &MessageInfo) -> StdResult<()> {
    let root = query_sudo_root(&deps.querier)?;
    if root.is_empty() || info.sender.as_str() != root {
        return Err(invalid("sender is not current x/sudo root"));
    }
    Ok(())
}
fn schedule(
    deps: DepsMut,
    env: Env,
    plan: ScheduleRequest,
) -> StdResult<Response> {
    if plan.name.is_empty() {
        return Err(invalid("upgrade name cannot be empty"));
    }
    if applied(deps.as_ref(), &plan.name)?.i64() != 0 {
        return Err(invalid("upgrade name has already been completed"));
    }
    match plan.target_time {
        Some(time) => {
            if plan.height.i64() != 0 {
                return Err(invalid(
                    "specify exactly one of target time or height",
                ));
            }
            if time.u64() <= env.block.time.nanos() {
                return Err(invalid("target time must be in the future"));
            }
        }
        None => {
            if plan.height.i64() <= 0
                || (plan.height.i64() as u64) < env.block.height
            {
                return Err(invalid("upgrade height must not be in the past"));
            }
        }
    }
    let id = NEXT_ID
        .load(deps.storage)?
        .checked_add(1)
        .ok_or_else(|| invalid("request ID overflow"))?;
    NEXT_ID.save(deps.storage, &id)?;
    PENDING.save(
        deps.storage,
        &PendingSchedule {
            id: Uint64::new(id),
            name: plan.name,
            height: if plan.target_time.is_none() {
                Some(plan.height)
            } else {
                None
            },
            target_time: plan.target_time,
            info: plan.info,
        },
    )?;
    Ok(Response::new().add_attribute("action", "schedule"))
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn sudo(deps: DepsMut, env: Env, msg: SudoMsg) -> StdResult<Response> {
    match msg {
        SudoMsg::SelectExecutionHeight {} => {
            let Some(mut pending) = PENDING.may_load(deps.storage)? else {
                return Ok(Response::new());
            };
            if pending.height.is_some() {
                return Ok(Response::new());
            }
            let target = pending
                .target_time
                .ok_or_else(|| invalid("pending time target missing"))?;
            if env.block.time.nanos() < target.u64() {
                return Ok(Response::new());
            }
            let delay = CONFIG.load(deps.storage)?.activation_delay_blocks.u64();
            if delay < 2 || delay > i64::MAX as u64 {
                return Err(invalid("invalid persisted activation delay"));
            }
            let height = env
                .block
                .height
                .checked_add(delay)
                .filter(|h| *h <= i64::MAX as u64)
                .ok_or_else(|| invalid("execution height overflow"))?;
            pending.height = Some(Int64::new(height as i64));
            PENDING.save(deps.storage, &pending)?;
            Ok(Response::new().add_event(
                Event::new("upgrade_execution_height_selected")
                    .add_attribute("name", pending.name)
                    .add_attribute("height", height.to_string())
                    .add_attribute("delay_blocks", delay.to_string()),
            ))
        }
        SudoMsg::SetModuleVersions { versions } => {
            set_versions(deps.storage, versions)?;
            Ok(Response::new())
        }
        SudoMsg::SetProtocolVersion { version } => {
            PROTOCOL_VERSION.save(deps.storage, &version)?;
            Ok(Response::new())
        }
        SudoMsg::RecordCompleted { name } => {
            if name.is_empty() || env.block.height > i64::MAX as u64 {
                return Err(invalid("invalid completed upgrade"));
            }
            let record = CompletedUpgrade {
                name: name.clone(),
                height: Int64::new(env.block.height as i64),
            };
            history().save(deps.storage, (env.block.height, name), &record)?;
            Ok(Response::new())
        }
        SudoMsg::ClearPlan {} => {
            PENDING.remove(deps.storage);
            Ok(Response::new())
        }
    }
}
fn set_versions(
    store: &mut dyn cosmwasm_std::Storage,
    versions: Vec<ModuleVersion>,
) -> StdResult<()> {
    for version in versions {
        if version.name.is_empty() {
            return Err(invalid("module name cannot be empty"));
        }
        MODULE_VERSIONS.save(store, &version.name, &version.version)?;
    }
    Ok(())
}
fn current_plan(deps: Deps) -> StdResult<Option<Plan>> {
    Ok(PENDING.may_load(deps.storage)?.and_then(|p| {
        p.height.map(|height| Plan {
            name: p.name,
            height,
            info: p.info,
        })
    }))
}
fn applied(deps: Deps, name: &str) -> StdResult<Int64> {
    let result = history()
        .idx
        .name
        .prefix(name.to_string())
        .range(deps.storage, None, None, Order::Ascending)
        .next()
        .transpose()?;
    Ok(result.map(|(_, r)| r.height).unwrap_or_else(Int64::zero))
}
fn modules(deps: Deps, name: Option<String>) -> StdResult<Vec<ModuleVersion>> {
    if let Some(name) = name {
        return Ok(MODULE_VERSIONS
            .may_load(deps.storage, &name)?
            .map(|version| vec![ModuleVersion { name, version }])
            .unwrap_or_default());
    }
    MODULE_VERSIONS
        .range(deps.storage, None, None, Order::Ascending)
        .map(|r| r.map(|(name, version)| ModuleVersion { name, version }))
        .collect()
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::CurrentPlan {} => to_json_binary(&current_plan(deps)?),
        QueryMsg::TimeTarget {} => to_json_binary(
            &PENDING
                .may_load(deps.storage)?
                .filter(|p| p.height.is_none())
                .and_then(|p| {
                    p.target_time.map(|target_time| TimeTarget {
                        id: p.id,
                        name: p.name,
                        target_time,
                    })
                }),
        ),
        QueryMsg::Status {} => to_json_binary(&PENDING.may_load(deps.storage)?),
        QueryMsg::Config {} => to_json_binary(&CONFIG.load(deps.storage)?),
        QueryMsg::Snapshot {} => to_json_binary(&Snapshot {
            plan: current_plan(deps)?,
            completed: history()
                .range(deps.storage, None, None, Order::Ascending)
                .map(|r| r.map(|(_, v)| v))
                .collect::<StdResult<_>>()?,
            module_versions: modules(deps, None)?,
            protocol_version: PROTOCOL_VERSION.may_load(deps.storage)?,
        }),
        QueryMsg::Applied { name } => to_json_binary(&applied(deps, &name)?),
        QueryMsg::LastCompleted {} => to_json_binary(
            &history()
                .range(deps.storage, None, None, Order::Descending)
                .next()
                .transpose()?
                .map(|(_, v)| v),
        ),
        QueryMsg::History {
            start_after,
            limit,
            descending,
        } => {
            let bound = start_after
                .map(|r| Bound::exclusive((r.height.i64() as u64, r.name)));
            let (min, max, order) = if descending.unwrap_or(false) {
                (None, bound, Order::Descending)
            } else {
                (bound, None, Order::Ascending)
            };
            let rows = history()
                .range(deps.storage, min, max, order)
                .take(limit.unwrap_or(50).clamp(1, 200) as usize)
                .map(|r| r.map(|(_, v)| v))
                .collect::<StdResult<Vec<_>>>()?;
            to_json_binary(&rows)
        }
        QueryMsg::ModuleVersions { name } => {
            to_json_binary(&modules(deps, name)?)
        }
        QueryMsg::ProtocolVersion {} => {
            to_json_binary(&PROTOCOL_VERSION.may_load(deps.storage)?)
        }
    }
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn migrate(
    _deps: DepsMut,
    _env: Env,
    _msg: MigrateMsg,
) -> StdResult<Response> {
    Ok(Response::new().add_attribute("action", "migrate"))
}
