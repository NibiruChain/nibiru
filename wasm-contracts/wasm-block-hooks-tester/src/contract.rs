use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response,
};

use crate::{
    msg::{ExecuteMsg, InstantiateMsg, QueryMsg, SudoMsg, WasmSudoMsg},
    state::{State, STATE},
};

type ContractError = anyhow::Error;

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    STATE.save(
        deps.storage,
        &State {
            count: 0,
            query_error: false,
            wasm_sudo_msg_calls: vec![],
            last_sudo: None,
        },
    )?;

    Ok(Response::default().add_attribute("method", "instantiate"))
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Config {
            count,
            query_error,
            wasm_sudo_msg_calls,
        } => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, ContractError> {
                    if let Some(count) = count {
                        state.count = count;
                    }
                    if let Some(query_error) = query_error {
                        state.query_error = query_error;
                    }
                    if let Some(wasm_sudo_msg_calls) = wasm_sudo_msg_calls {
                        state.wasm_sudo_msg_calls = wasm_sudo_msg_calls;
                    }
                    Ok(state)
                },
            )?;
            Ok(Response::default().add_attribute("method", "config"))
        }
    }
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(
    deps: Deps,
    env: Env,
    msg: QueryMsg,
) -> Result<Binary, ContractError> {
    match msg {
        QueryMsg::BeginBlockPlan {} | QueryMsg::EndBlockPlan {} => {
            to_json_binary(&query_plan(deps, env)?).map_err(ContractError::from)
        }
        QueryMsg::State {} => to_json_binary(&STATE.load(deps.storage)?)
            .map_err(ContractError::from),
    }
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn sudo(
    deps: DepsMut,
    _env: Env,
    msg: SudoMsg,
) -> Result<Response, ContractError> {
    match msg {
        SudoMsg::Increment { by } => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, ContractError> {
                    state.count += by;
                    state.last_sudo = Some("increment".to_string());
                    Ok(state)
                },
            )?;
            Ok(Response::default()
                .add_attribute("method", "sudo_increment")
                .add_attribute("by", by.to_string()))
        }
        SudoMsg::Set { count } => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, ContractError> {
                    state.count = count;
                    state.last_sudo = Some("set".to_string());
                    Ok(state)
                },
            )?;
            Ok(Response::default()
                .add_attribute("method", "sudo_set")
                .add_attribute("count", count.to_string()))
        }
        SudoMsg::FailBeforeWrite {} => {
            anyhow::bail!("fixture sudo failure before write")
        }
        SudoMsg::FailAfterWrite { by } => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, ContractError> {
                    state.count += by;
                    state.last_sudo = Some("fail_after_write".to_string());
                    Ok(state)
                },
            )?;
            anyhow::bail!("fixture sudo failure after write")
        }
    }
}

fn query_plan(deps: Deps, _env: Env) -> Result<Vec<WasmSudoMsg>, ContractError> {
    let state = STATE.load(deps.storage)?;
    if state.query_error {
        anyhow::bail!("fixture registry query error");
    }
    Ok(state.wasm_sudo_msg_calls)
}

#[cfg(test)]
mod tests {
    use cosmwasm_std::{
        from_json,
        testing::{mock_dependencies, mock_env, mock_info},
    };

    use crate::{
        contract::{execute, instantiate, query, sudo},
        msg::{ExecuteMsg, InstantiateMsg, QueryMsg, SudoMsg, WasmSudoMsg},
        state::State,
    };

    const SENDER: &str = "sender";
    const TARGET: &str = "target_contract";

    fn setup() -> anyhow::Result<(
        cosmwasm_std::OwnedDeps<
            cosmwasm_std::testing::MockStorage,
            cosmwasm_std::testing::MockApi,
            cosmwasm_std::testing::MockQuerier,
        >,
        cosmwasm_std::Env,
    )> {
        let mut deps = mock_dependencies();
        let env = mock_env();
        instantiate(
            deps.as_mut(),
            env.clone(),
            mock_info(SENDER, &[]),
            InstantiateMsg {},
        )?;
        Ok((deps, env))
    }

    fn query_state(
        deps: cosmwasm_std::Deps,
        env: cosmwasm_std::Env,
    ) -> anyhow::Result<State> {
        Ok(from_json(query(deps, env, QueryMsg::State {})?)?)
    }

    #[test]
    fn instantiate_starts_from_blank_state() -> anyhow::Result<()> {
        let (deps, env) = setup()?;

        let state = query_state(deps.as_ref(), env.clone())?;
        assert_eq!(state.count, 0);
        assert!(!state.query_error);
        assert!(state.wasm_sudo_msg_calls.is_empty());
        assert_eq!(state.last_sudo, None);

        let calls: Vec<WasmSudoMsg> =
            from_json(query(deps.as_ref(), env, QueryMsg::BeginBlockPlan {})?)?;

        assert!(calls.is_empty());
        Ok(())
    }

    #[test]
    fn config_updates_wasm_sudo_msg_calls() -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;
        let calls = vec![
            WasmSudoMsg {
                contract_addr: TARGET.to_string(),
                msg: serde_json::to_value(SudoMsg::Increment { by: 7 })?,
            },
            WasmSudoMsg {
                contract_addr: "target_two".to_string(),
                msg: serde_json::to_value(SudoMsg::FailAfterWrite { by: 9 })?,
            },
        ];

        execute(
            deps.as_mut(),
            env.clone(),
            mock_info(SENDER, &[]),
            ExecuteMsg::Config {
                count: Some(42),
                query_error: Some(false),
                wasm_sudo_msg_calls: Some(calls.clone()),
            },
        )?;

        let state = query_state(deps.as_ref(), env.clone())?;
        assert_eq!(state.count, 42);
        assert_eq!(state.wasm_sudo_msg_calls, calls);

        let queried_calls: Vec<WasmSudoMsg> =
            from_json(query(deps.as_ref(), env, QueryMsg::EndBlockPlan {})?)?;

        assert_eq!(queried_calls, calls);
        let sudo_msg: SudoMsg =
            serde_json::from_value(queried_calls[0].msg.clone())?;
        assert_eq!(sudo_msg, SudoMsg::Increment { by: 7 });
        Ok(())
    }

    #[test]
    fn query_error_config_makes_registry_queries_fail() -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;

        execute(
            deps.as_mut(),
            env.clone(),
            mock_info(SENDER, &[]),
            ExecuteMsg::Config {
                count: None,
                query_error: Some(true),
                wasm_sudo_msg_calls: None,
            },
        )?;

        let err = query(deps.as_ref(), env, QueryMsg::BeginBlockPlan {})
            .expect_err("query should fail");

        assert!(err.to_string().contains("fixture registry query error"));
        Ok(())
    }

    #[test]
    fn sudo_increment_mutates_counter() -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;

        let res =
            sudo(deps.as_mut(), env.clone(), SudoMsg::Increment { by: 5 })?;

        assert_eq!(res.attributes[0].value, "sudo_increment");
        let state = query_state(deps.as_ref(), env)?;
        assert_eq!(state.count, 5);
        assert_eq!(state.last_sudo, Some("increment".to_string()));
        Ok(())
    }

    #[test]
    fn sudo_fail_before_write_leaves_counter_unchanged() -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;

        let err = sudo(deps.as_mut(), env.clone(), SudoMsg::FailBeforeWrite {})
            .expect_err("sudo should fail");

        assert!(err.to_string().contains("failure before write"));
        let state = query_state(deps.as_ref(), env)?;
        assert_eq!(state.count, 0);
        assert_eq!(state.last_sudo, None);
        Ok(())
    }

    #[test]
    fn sudo_fail_after_write_mutates_before_returning_error(
    ) -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;

        let err = sudo(
            deps.as_mut(),
            env.clone(),
            SudoMsg::FailAfterWrite { by: 9 },
        )
        .expect_err("sudo should fail");

        assert!(err.to_string().contains("failure after write"));
        let state = query_state(deps.as_ref(), env)?;
        assert_eq!(state.count, 9);
        assert_eq!(state.last_sudo, Some("fail_after_write".to_string()));
        Ok(())
    }

    #[test]
    fn config_can_replace_wasm_sudo_msg_calls() -> anyhow::Result<()> {
        let (mut deps, env) = setup()?;
        let first_calls = vec![WasmSudoMsg {
            contract_addr: TARGET.to_string(),
            msg: serde_json::to_value(SudoMsg::Increment { by: 7 })?,
        }];
        let second_calls = vec![WasmSudoMsg {
            contract_addr: "other_target".to_string(),
            msg: serde_json::to_value(SudoMsg::FailBeforeWrite {})?,
        }];

        execute(
            deps.as_mut(),
            env.clone(),
            mock_info(SENDER, &[]),
            ExecuteMsg::Config {
                count: None,
                query_error: None,
                wasm_sudo_msg_calls: Some(first_calls),
            },
        )?;
        execute(
            deps.as_mut(),
            env.clone(),
            mock_info(SENDER, &[]),
            ExecuteMsg::Config {
                count: None,
                query_error: None,
                wasm_sudo_msg_calls: Some(second_calls.clone()),
            },
        )?;

        let calls: Vec<WasmSudoMsg> =
            from_json(query(deps.as_ref(), env, QueryMsg::BeginBlockPlan {})?)?;
        assert_eq!(calls, second_calls);
        Ok(())
    }

    #[test]
    fn golden_json_payloads_are_stable() -> anyhow::Result<()> {
        let begin_block = serde_json::to_string(&QueryMsg::BeginBlockPlan {})?;
        let end_block = serde_json::to_string(&QueryMsg::EndBlockPlan {})?;
        let instantiate = serde_json::to_string(&InstantiateMsg {})?;
        let increment = serde_json::to_string(&SudoMsg::Increment { by: 7 })?;
        let set = serde_json::to_string(&SudoMsg::Set { count: 42 })?;
        let fail_before = serde_json::to_string(&SudoMsg::FailBeforeWrite {})?;
        let fail_after =
            serde_json::to_string(&SudoMsg::FailAfterWrite { by: 9 })?;
        let wasm_sudo_msg = serde_json::to_string(&WasmSudoMsg {
            contract_addr: TARGET.to_string(),
            msg: serde_json::to_value(SudoMsg::Increment { by: 7 })?,
        })?;
        let config = serde_json::to_string(&ExecuteMsg::Config {
            count: Some(42),
            query_error: Some(false),
            wasm_sudo_msg_calls: Some(vec![WasmSudoMsg {
                contract_addr: TARGET.to_string(),
                msg: serde_json::to_value(SudoMsg::FailAfterWrite { by: 9 })?,
            }]),
        })?;

        println!("begin_block_query={begin_block}");
        println!("end_block_query={end_block}");
        println!("instantiate={instantiate}");
        println!("config={config}");
        println!("sudo_increment={increment}");
        println!("sudo_set={set}");
        println!("sudo_fail_before_write={fail_before}");
        println!("sudo_fail_after_write={fail_after}");
        println!("wasm_sudo_msg={wasm_sudo_msg}");

        assert_eq!(begin_block, r#"{"begin_block_plan":{}}"#);
        assert_eq!(end_block, r#"{"end_block_plan":{}}"#);
        assert_eq!(instantiate, r#"{}"#);
        assert_eq!(
            config,
            r#"{"config":{"count":42,"query_error":false,"wasm_sudo_msg_calls":[{"contract_addr":"target_contract","msg":{"fail_after_write":{"by":9}}}]}}"#
        );
        assert_eq!(increment, r#"{"increment":{"by":7}}"#);
        assert_eq!(set, r#"{"set":{"count":42}}"#);
        assert_eq!(fail_before, r#"{"fail_before_write":{}}"#);
        assert_eq!(fail_after, r#"{"fail_after_write":{"by":9}}"#);
        assert_eq!(
            wasm_sudo_msg,
            r#"{"contract_addr":"target_contract","msg":{"increment":{"by":7}}}"#
        );
        Ok(())
    }
}
