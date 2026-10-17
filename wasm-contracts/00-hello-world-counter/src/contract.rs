use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response,
};
use cw2::set_contract_version;

use crate::{
    msg::{ExecuteMsg, InstantiateMsg, QueryMsg},
    state::{State, STATE},
};

type ContractError = anyhow::Error;

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(
        deps.storage,
        format!("nibiru-wasm/contracts/{CONTRACT_NAME}"),
        CONTRACT_VERSION,
    )?;

    STATE.save(
        deps.storage,
        &State {
            count: msg.count,
            owner: info.sender.clone(),
        },
    )?;
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Increment {} => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, anyhow::Error> {
                    state.count += 1;
                    Ok(state)
                },
            )?;
            Ok(Response::default())
        }
        ExecuteMsg::Reset { count } => {
            STATE.update(
                deps.storage,
                |mut state| -> Result<_, anyhow::Error> {
                    let owner = state.owner.clone();
                    if info.sender != owner {
                        return Err(anyhow::anyhow!(
                            "Unauthorized: only the owner ({owner}) can use reset",
                        ));
                    }
                    state.count = count;
                    Ok(state)
                },
            )?;
            Ok(Response::default())
        }
    }
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(
    deps: Deps,
    _env: Env,
    msg: QueryMsg,
) -> Result<Binary, ContractError> {
    match msg {
        QueryMsg::Count {} => {
            let state = STATE.load(deps.storage)?;
            Ok(to_json_binary(&state)?)
        }
    }
}

pub const CONTRACT_NAME: &str = env!("CARGO_PKG_NAME");
pub const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg(test)]
pub mod tests {

    use easy_addr;
    use nibiru_std::errors::TestResult;

    use crate::{
        contract::{execute, query},
        msg::{ExecuteMsg, QueryMsg},
        state::State,
        tutil::{mock_info_for_sender, setup_contract, TEST_OWNER},
    };

    struct TestCaseExec<'a> {
        exec_msg: ExecuteMsg,
        sender: &'a str,
        err: Option<&'a str>,
        start_count: i64,
        want_count_after: i64,
    }

    /// Test that all owner-gated execute calls fail when the tx sender is not
    /// the smart contract owner.
    #[test]
    pub fn test_exec() -> TestResult {
        let not_owner = easy_addr::addr!("not-owner");

        let test_cases: Vec<TestCaseExec> = vec![
            TestCaseExec {
                sender: not_owner,
                exec_msg: ExecuteMsg::Increment {},
                err: None,
                start_count: 0,
                want_count_after: 1,
            },
            TestCaseExec {
                sender: not_owner,
                exec_msg: ExecuteMsg::Increment {},
                err: None,
                start_count: -70,
                want_count_after: -69,
            },
            TestCaseExec {
                sender: TEST_OWNER,
                exec_msg: ExecuteMsg::Reset { count: 25 },
                err: None,
                start_count: i64::MAX,
                want_count_after: 25,
            },
            TestCaseExec {
                sender: TEST_OWNER,
                exec_msg: ExecuteMsg::Reset { count: -25 },
                err: None,
                start_count: 0,
                want_count_after: -25,
            },
            TestCaseExec {
                sender: not_owner,
                exec_msg: ExecuteMsg::Reset { count: 25 },
                err: Some("Unauthorized: only the owner"),
                start_count: 0,      // unused
                want_count_after: 0, // unused
            },
        ];

        for tc in &test_cases {
            // instantiate smart contract from the owner
            let (mut deps, env, _info) = setup_contract(tc.start_count)?;

            // send the exec msg and it should fail.
            let info = mock_info_for_sender(tc.sender);
            let res =
                execute(deps.as_mut(), env.clone(), info, tc.exec_msg.clone());

            if let Some(want_err) = tc.err {
                let err = res.expect_err("err should be defined");
                let is_contained = err.to_string().contains(want_err);
                assert!(is_contained, "got error {}", err);
                continue;
            }

            let res = res?;
            assert_eq!(res.messages.len(), 0);

            let query_req = QueryMsg::Count {};
            let binary = query(deps.as_ref(), env, query_req)?;
            let query_resp: State = cosmwasm_std::from_json(binary)?;

            let state = query_resp;
            let got_count_after = state.count;
            assert_eq!(got_count_after, tc.want_count_after);
            assert_eq!(state.owner.as_str(), TEST_OWNER);
        }
        Ok(())
    }
}
