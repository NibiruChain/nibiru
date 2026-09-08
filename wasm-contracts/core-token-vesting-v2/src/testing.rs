use crate::contract::{execute, instantiate, query};
use crate::errors::{ContractError, VestingError};
use crate::msg::{
    DeregisterUserResponse, ExecuteMsg, InstantiateMsg, QueryMsg,
    RewardUserRequest, VestingAccountResponse, VestingData, VestingSchedule,
    VestingScheduleQueryOutput,
};

use cosmwasm_std::testing::{MockApi, MockQuerier, MockStorage};
use cosmwasm_std::{coin, testing, Empty, MessageInfo};
use cosmwasm_std::{
    from_json,
    testing::{mock_dependencies, mock_env, mock_info},
    Attribute, BankMsg, Coin, Env, OwnedDeps, Response, StdError, SubMsg,
    Timestamp, Uint128, Uint64,
};

pub type TestResult = Result<(), anyhow::Error>;

pub fn mock_env_with_time(block_time: u64) -> Env {
    let mut env = testing::mock_env();
    env.block.time = Timestamp::from_seconds(block_time);
    env
}

/// Convenience function for instantiating the contract at and setting up
/// the env to have the given block time.
pub fn setup_with_block_time(
    block_time: u64,
) -> anyhow::Result<(OwnedDeps<MockStorage, MockApi, MockQuerier, Empty>, Env)> {
    let mut deps = testing::mock_dependencies();
    let env = mock_env_with_time(block_time);
    instantiate(
        deps.as_mut(),
        env.clone(),
        testing::mock_info("admin-sender", &[coin(5000, "token")]),
        InstantiateMsg {
            admin: "admin-sender".to_string(),
            managers: vec!["manager-sender".to_string()],
        },
    )?;
    Ok((deps, env))
}

#[test]
fn proper_initialization() -> TestResult {
    let mut deps = mock_dependencies();

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec!["admin-sender".to_string()],
    };

    let info = mock_info("addr0000", &[coin(1000, "nibi")]);

    let _res = instantiate(deps.as_mut(), mock_env(), info, msg)?;
    Ok(())
}

#[test]
fn invalid_coin_sent_instantiation() -> TestResult {
    let mut deps = mock_dependencies();

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec!["admin-sender".to_string()],
    };

    // No coins sent
    let res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        msg.clone(),
    );
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "must deposit exactly one type of token".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    // 2 coins sent
    let res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(1000, "nibi"), coin(1000, "usd")]),
        msg.clone(),
    );
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "must deposit exactly one type of token".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    // 0 amount coins sent
    let res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(0, "nibi")]),
        msg,
    );
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "must deposit some token".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    Ok(())
}

#[test]
fn invalid_manangers_initialization() -> TestResult {
    let mut deps = mock_dependencies();

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec![],
    };

    let info = mock_info("addr0000", &[coin(1000, "nibi")]);

    let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg.clone());
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "managers cannot be empty".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec!["".to_string()],
    };
    let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg.clone());
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "Invalid input: human address too short for this mock implementation (must be >= 3).".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec!["admin-sender".to_string(), "".to_string()],
    };
    let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg.clone());
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "Invalid input: human address too short for this mock implementation (must be >= 3).".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    let msg = InstantiateMsg {
        admin: "".to_string(),
        managers: vec!["admin-sender".to_string()],
    };
    let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg.clone());
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "Invalid input: human address too short for this mock implementation (must be >= 3).".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    Ok(())
}

#[test]
fn invalid_managers() -> TestResult {
    let mut deps = mock_dependencies();

    let msg = InstantiateMsg {
        admin: "admin-sender".to_string(),
        managers: vec!["admin-manager".to_string()],
    };

    // No coins sent
    let res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        msg.clone(),
    );
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "must deposit exactly one type of token".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    // 2 coins sent
    let res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(1000, "nibi"), coin(1000, "usd")]),
        msg,
    );
    match res {
        Err(err) => {
            assert_eq!(
                err,
                StdError::GenericErr {
                    msg: "must deposit exactly one type of token".to_string(),
                }
            )
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }

    Ok(())
}

#[test]
fn register_cliff_vesting_account_with_native_token() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(2000, "uusd")]),
        InstantiateMsg {
            admin: "addr0000".to_string(),
            managers: vec!["admin-sender".to_string()],
        },
    )?;

    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    let create_msg = |start_time: u64,
                      end_time: u64,
                      vesting_amount: u128,
                      cliff_amount: u128,
                      cliff_time: u64|
     -> ExecuteMsg {
        ExecuteMsg::RewardUsers {
            rewards: vec![RewardUserRequest {
                user_address: "addr0001".to_string(),
                vesting_amount: Uint128::new(vesting_amount),
                cliff_amount: Uint128::new(cliff_amount),
            }],
            vesting_schedule: VestingSchedule::LinearVestingWithCliff {
                start_time: Uint64::new(start_time),
                end_time: Uint64::new(end_time),
                cliff_time: Uint64::new(cliff_time),
            },
        }
    };

    // unauthorized sender
    let msg = create_msg(100, 110, 0, 1000, 105);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0042", &[]),
        msg,
        StdError::generic_err(
            "Sender addr0042 is unauthorized to reward users.",
        )
        .into(),
    );

    // zero amount vesting token
    let msg = create_msg(100, 110, 0, 1000, 105);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[]),
        msg,
        ContractError::Vesting(VestingError::ZeroVestingAmount),
    );

    // cliff time less than block time
    let msg = create_msg(100, 110, 1000, 500, 99);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(1000u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::InvalidTimeRange {
            start_time: 100,
            cliff_time: 99,
            end_time: 110,
        }),
    );

    // end time less than start time
    let msg = create_msg(110, 100, 1000, 1000, 105);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(1000u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::InvalidTimeRange {
            start_time: 110,
            cliff_time: 105,
            end_time: 100,
        }),
    );

    // cliff amount greater than vesting amount
    let (vesting_amount, cliff_amount, cliff_time) = (1000, 1001, 105);
    let msg = create_msg(100, 110, vesting_amount, cliff_amount, cliff_time);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(1000u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::ExcessiveAmount {
            cliff_amount,
            vesting_amount,
        }),
    );

    // deposit amount higher than unallocated
    let (vesting_amount, cliff_amount, cliff_time) = (10000, 250, 105);
    let msg = create_msg(100, 110, vesting_amount, cliff_amount, cliff_time);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(999u128, "uusd")]),
        msg,
        StdError::generic_err(
            "Insufficient funds for all rewards. Contract has 2000 available but trying to allocate 10000",
        )
        .into(),
    );

    // valid amount
    let (vesting_amount, cliff_amount, cliff_time) = (1000, 250, 105);
    let msg = create_msg(100, 110, vesting_amount, cliff_amount, cliff_time);

    let res =
        execute(deps.as_mut(), env.clone(), mock_info("addr0000", &[]), msg)?;

    assert_eq!(
        res.attributes,
        vec![
            Attribute {
                key: "action".to_string(),
                value: "register_vesting_account".to_string()
            },
            Attribute {
                key: "address".to_string(),
                value: "addr0001".to_string()
            },
            Attribute {
                key: "vesting_amount".to_string(),
                value: "1000".to_string()
            },
            Attribute {
                key: "method".to_string(),
                value: "reward_users".to_string()
            }
        ]
    );

    // valid amount - one failed because duplicate
    let vesting_amount = 500u128;
    let cliff_amount = 250u128;
    let cliff_time = 105u64;

    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![
            RewardUserRequest {
                user_address: "addr0002".to_string(),
                vesting_amount: Uint128::new(vesting_amount),
                cliff_amount: Uint128::new(cliff_amount),
            },
            RewardUserRequest {
                user_address: "addr0002".to_string(),
                vesting_amount: Uint128::new(vesting_amount),
                cliff_amount: Uint128::new(cliff_amount),
            },
        ],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(cliff_time),
        },
    };

    let res =
        execute(deps.as_mut(), env.clone(), mock_info("addr0000", &[]), msg)?;

    assert_eq!(
        res.attributes,
        vec![
            Attribute {
                key: "action".to_string(),
                value: "register_vesting_account".to_string()
            },
            Attribute {
                key: "address".to_string(),
                value: "addr0002".to_string()
            },
            Attribute {
                key: "vesting_amount".to_string(),
                value: "500".to_string()
            },
            Attribute {
                key: "method".to_string(),
                value: "reward_users".to_string()
            }
        ]
    );

    Ok(())
}

#[test]
fn test_withdraw() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(2000, "uusd")]),
        InstantiateMsg {
            admin: "addr0000".to_string(),
            managers: vec!["admin-sender".to_string()],
        },
    )?;

    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    let create_msg = |start_time: u64,
                      end_time: u64,
                      vesting_amount: u128,
                      cliff_amount: u128,
                      cliff_time: u64|
     -> ExecuteMsg {
        ExecuteMsg::RewardUsers {
            rewards: vec![RewardUserRequest {
                user_address: "addr0001".to_string(),
                vesting_amount: Uint128::new(vesting_amount),
                cliff_amount: Uint128::new(cliff_amount),
            }],
            vesting_schedule: VestingSchedule::LinearVestingWithCliff {
                start_time: Uint64::new(start_time),
                end_time: Uint64::new(end_time),
                cliff_time: Uint64::new(cliff_time),
            },
        }
    };

    // valid amount
    let (vesting_amount, cliff_amount, cliff_time) = (1000, 250, 105);
    let msg = create_msg(100, 110, vesting_amount, cliff_amount, cliff_time);

    let _res =
        execute(deps.as_mut(), env.clone(), mock_info("addr0000", &[]), msg)?;

    // try to withdraw

    // unauthorized sender
    let msg = ExecuteMsg::Withdraw {
        amount: Uint128::new(1000),
    };
    require_error(
        &mut deps,
        &env,
        mock_info("addr0042", &[]),
        msg,
        StdError::generic_err("Unauthorized").into(),
    );

    // withdraw more than unallocated
    let msg = ExecuteMsg::Withdraw {
        amount: Uint128::new(1001),
    };
    let res =
        execute(deps.as_mut(), env.clone(), mock_info("addr0000", &[]), msg)?;

    assert_eq!(
        res.attributes,
        vec![
            Attribute {
                key: "action".to_string(),
                value: "withdraw".to_string()
            },
            Attribute {
                key: "recipient".to_string(),
                value: "addr0000".to_string()
            },
            Attribute {
                key: "amount".to_string(),
                value: "1000".to_string()
            },
            Attribute {
                key: "unallocated_amount".to_string(),
                value: "0".to_string()
            },
        ]
    );

    // withdraw but there's no more unallocated
    let msg = ExecuteMsg::Withdraw {
        amount: Uint128::new(1),
    };
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[]),
        msg,
        StdError::generic_err("Nothing to withdraw").into(),
    );

    Ok(())
}

fn require_error(
    deps: &mut OwnedDeps<MockStorage, MockApi, MockQuerier>,
    env: &Env,
    info: MessageInfo,
    msg: ExecuteMsg,
    expected_error: ContractError,
) {
    let res = execute(deps.as_mut(), env.clone(), info, msg);
    match res {
        Err(err) => {
            assert_eq!(err, expected_error)
        }
        Ok(_) => panic!("Expected error but got success: {res:?}"),
    }
}

#[test]
fn register_vesting_account_with_native_token() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(1000, "uusd")]),
        InstantiateMsg {
            admin: "addr0000".to_string(),
            managers: vec!["admin-sender".to_string()],
        },
    )?;

    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // zero amount vesting token
    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![RewardUserRequest {
            user_address: "addr0001".to_string(),
            vesting_amount: Uint128::zero(),
            cliff_amount: Uint128::zero(),
        }],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(105),
        },
    };

    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(0u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::ZeroVestingAmount),
    );

    // too much vesting amount
    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![RewardUserRequest {
            user_address: "addr0001".to_string(),
            vesting_amount: Uint128::new(1000001u128),
            cliff_amount: Uint128::zero(),
        }],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(105),
        },
    };
    let info = mock_info("addr0000", &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg.clone());
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => {
            assert_eq!(
                msg,
                "Insufficient funds for all rewards. Contract has 1000 available but trying to allocate 1000001"
            )
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // too much vesting amount in 2 rewards
    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![
            RewardUserRequest {
                user_address: "addr0001".to_string(),
                vesting_amount: Uint128::new(1000u128),
                cliff_amount: Uint128::zero(),
            },
            RewardUserRequest {
                user_address: "addr0001".to_string(),
                vesting_amount: Uint128::new(1u128),
                cliff_amount: Uint128::zero(),
            },
        ],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(105),
        },
    };
    let info = mock_info("addr0000", &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg.clone());
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => {
            assert_eq!(
                msg,
                "Insufficient funds for all rewards. Contract has 1000 available but trying to allocate 1001"
            )
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // valid amount
    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![RewardUserRequest {
            user_address: "addr0001".to_string(),
            vesting_amount: Uint128::new(100u128),
            cliff_amount: Uint128::zero(),
        }],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(105),
        },
    };
    let info = mock_info("addr0000", &[Coin::new(1000u128, "uusd")]);
    let res: Response = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.attributes,
        vec![
            Attribute {
                key: "action".to_string(),
                value: "register_vesting_account".to_string()
            },
            Attribute {
                key: "address".to_string(),
                value: "addr0001".to_string()
            },
            Attribute {
                key: "vesting_amount".to_string(),
                value: "100".to_string()
            },
            Attribute {
                key: "method".to_string(),
                value: "reward_users".to_string()
            }
        ]
    );

    // query vesting account
    assert_eq!(
        from_json::<VestingAccountResponse>(&query(
            deps.as_ref(),
            env,
            QueryMsg::VestingAccount {
                address: "addr0001".to_string(),
                start_after: None,
                limit: None,
            },
        )?)?,
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![VestingData {
                master_address: Some("addr0000".to_string()),
                vesting_amount: Uint128::new(100u128),
                vesting_schedule:
                    VestingScheduleQueryOutput::LinearVestingWithCliff {
                        start_time: Uint64::new(100),
                        end_time: Uint64::new(110),
                        cliff_time: Uint64::new(105),
                        vesting_amount: Uint128::new(100u128),
                        cliff_amount: Uint128::zero(),
                    },
                vesting_denom: cw20::Denom::Native("uusd".to_string()),
                vested_amount: Uint128::zero(),
                claimable_amount: Uint128::zero(),
            }]
        },
    );
    Ok(())
}

#[test]
fn claim_native() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[coin(1000000u128, "uusd")]),
        InstantiateMsg {
            admin: "addr0000".to_string(),
            managers: vec!["admin-sender".to_string()],
        },
    )?;

    // init env to time 100
    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // valid amount
    let msg = ExecuteMsg::RewardUsers {
        rewards: vec![RewardUserRequest {
            user_address: "addr0001".to_string(),
            vesting_amount: Uint128::new(1000000u128),
            cliff_amount: Uint128::new(500000u128),
        }],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            cliff_time: Uint64::new(105),
            end_time: Uint64::new(110),
        },
    };

    let info = mock_info("addr0000", &[Coin::new(1000000u128, "uusd")]);
    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.attributes,
        vec![
            Attribute {
                key: "action".to_string(),
                value: "register_vesting_account".to_string()
            },
            Attribute {
                key: "address".to_string(),
                value: "addr0001".to_string()
            },
            Attribute {
                key: "vesting_amount".to_string(),
                value: "1000000".to_string()
            },
            Attribute {
                key: "method".to_string(),
                value: "reward_users".to_string()
            }
        ]
    );

    // make time to half claimable
    env.block.time = Timestamp::from_seconds(105);

    // valid claim
    let info = mock_info("addr0001", &[]);
    let msg = ExecuteMsg::Claim {};

    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg.clone())?;
    assert_eq!(
        res.messages,
        vec![SubMsg::new(BankMsg::Send {
            to_address: "addr0001".to_string(),
            amount: vec![Coin {
                denom: "uusd".to_string(),
                amount: Uint128::new(500000u128),
            }],
        }),]
    );
    assert_eq!(
        res.attributes,
        vec![
            Attribute::new("action", "claim"),
            Attribute::new("address", "addr0001"),
            Attribute::new("vesting_amount", "1000000"),
            Attribute::new("vested_amount", "500000"),
            Attribute::new("claim_amount", "500000"),
        ],
    );

    assert_eq!(
        from_json::<VestingAccountResponse>(&query(
            deps.as_ref(),
            env.clone(),
            QueryMsg::VestingAccount {
                address: "addr0001".to_string(),
                start_after: None,
                limit: None,
            },
        )?)?,
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![VestingData {
                master_address: Some("addr0000".to_string()),
                vesting_amount: Uint128::new(1000000u128),
                vesting_schedule:
                    VestingScheduleQueryOutput::LinearVestingWithCliff {
                        start_time: Uint64::new(100),
                        end_time: Uint64::new(110),
                        cliff_time: Uint64::new(105),
                        vesting_amount: Uint128::new(1000000u128),
                        cliff_amount: Uint128::new(500000u128),
                    },
                vesting_denom: cw20::Denom::Native("uusd".to_string()),
                vested_amount: Uint128::new(500000u128),
                claimable_amount: Uint128::zero(),
            }]
        },
    );

    // make time to half claimable
    env.block.time = Timestamp::from_seconds(110);

    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.messages,
        vec![SubMsg::new(BankMsg::Send {
            to_address: "addr0001".to_string(),
            amount: vec![Coin {
                denom: "uusd".to_string(),
                amount: Uint128::new(500000u128),
            }],
        }),]
    );
    assert_eq!(
        res.attributes,
        vec![
            Attribute::new("action", "claim"),
            Attribute::new("address", "addr0001"),
            Attribute::new("vesting_amount", "1000000"),
            Attribute::new("vested_amount", "1000000"),
            Attribute::new("claim_amount", "500000"),
        ],
    );

    // query vesting account
    assert_eq!(
        from_json::<VestingAccountResponse>(&query(
            deps.as_ref(),
            env,
            QueryMsg::VestingAccount {
                address: "addr0001".to_string(),
                start_after: None,
                limit: None,
            },
        )?)?,
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![],
        }
    );

    Ok(())
}

#[test]
fn deregister_err_nonexistent_vesting_account() -> TestResult {
    let (mut deps, env) = setup_with_block_time(50)?;

    let msg = ExecuteMsg::DeregisterVestingAccounts {
        addresses: vec!["nonexistent".to_string()],
    };
    let res = execute(
        deps.as_mut(),
        env, // Use the custom environment with the adjusted block time
        testing::mock_info("manager-sender", &[]),
        msg,
    )?;
    let response_items: Vec<DeregisterUserResponse> =
        from_json(res.data.unwrap()).unwrap();
    assert!(!response_items[0].success);
    let error_msg = response_items[0].clone().error_msg;
    if !error_msg.contains("Failed to deregister vesting account: Generic error: User nonexistent does not have a vesting account.") {
        panic!("Unexpected error message {error_msg:?}")
    }
    Ok(())
}

#[test]
fn deregister_err_unauthorized_vesting_account() -> TestResult {
    // Set up the environment with a block time before the vesting start time
    let (mut deps, env) = setup_with_block_time(50)?;

    // Try to deregister with unauthorized sender
    let msg = ExecuteMsg::DeregisterVestingAccounts {
        addresses: vec!["addr0001".to_string()],
    };
    require_error(
        &mut deps,
        &env,
        mock_info("addr0042", &[]),
        msg,
        StdError::generic_err(
            "Sender addr0042 is not authorized to deregister vesting accounts.",
        )
        .into(),
    );
    Ok(())
}

#[test]
fn deregister_successful() -> TestResult {
    // Set up the environment with a block time before the vesting start time
    let (mut deps, env) = setup_with_block_time(105)?;

    execute(
        deps.as_mut(),
        env.clone(), // Use the custom environment with the adjusted block time
        testing::mock_info("admin-sender", &[]),
        ExecuteMsg::RewardUsers {
            rewards: vec![RewardUserRequest {
                user_address: "addr0001".to_string(),
                vesting_amount: Uint128::new(5000u128),
                cliff_amount: Uint128::new(1250u128),
            }],
            vesting_schedule: VestingSchedule::LinearVestingWithCliff {
                start_time: Uint64::new(100),
                cliff_time: Uint64::new(105),
                end_time: Uint64::new(110),
            },
        },
    )?;

    // claim some of it
    execute(
        deps.as_mut(),
        env.clone(),
        testing::mock_info("addr0001", &[]),
        ExecuteMsg::Claim {},
    )?;

    // Deregister with the manager address
    let res = execute(
        deps.as_mut(),
        env, // Use the custom environment with the adjusted block time
        testing::mock_info("manager-sender", &[]),
        ExecuteMsg::DeregisterVestingAccounts {
            addresses: vec!["addr0001".to_string()],
        },
    )?;
    let data =
        from_json::<Vec<DeregisterUserResponse>>(res.data.unwrap()).unwrap();

    assert_eq!(
        data[0],
        DeregisterUserResponse {
            user_address: "addr0001".to_string(),
            success: true,
            error_msg: "".to_string(),
        }
    );
    assert_eq!(res.messages.len(), 1);
    assert_eq!(
        res.messages[0],
        SubMsg::new(BankMsg::Send {
            to_address: "admin-sender".to_string(),
            amount: vec![Coin {
                denom: "token".to_string(),
                amount: Uint128::new(3750u128),
            }],
        })
    );
    Ok(())
}

#[test]
fn query_vesting_accounts() -> TestResult {
    // Set up the environment with a block time before the vesting start time
    let (mut deps, env) = setup_with_block_time(105)?;

    let register_msg = ExecuteMsg::RewardUsers {
        rewards: vec![RewardUserRequest {
            user_address: "addr0001".to_string(),
            vesting_amount: Uint128::new(5000u128),
            cliff_amount: Uint128::new(1250u128),
        }],
        vesting_schedule: VestingSchedule::LinearVestingWithCliff {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            cliff_time: Uint64::new(105),
        },
    };

    execute(
        deps.as_mut(),
        env.clone(), // Use the custom environment with the adjusted block time
        testing::mock_info("admin-sender", &[]),
        register_msg,
    )?;

    let res = query(
        deps.as_ref(),
        env,
        QueryMsg::VestingAccounts {
            address: vec!["addr0001".to_string()],
        },
    )?;

    let response_items: Vec<VestingAccountResponse> = from_json(res).unwrap();
    assert_eq!(
        response_items[0],
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![VestingData {
                master_address: Some("admin-sender".to_string()),
                vesting_amount: Uint128::new(5000u128),
                vesting_schedule:
                    VestingScheduleQueryOutput::LinearVestingWithCliff {
                        start_time: Uint64::new(100),
                        end_time: Uint64::new(110),
                        cliff_time: Uint64::new(105),
                        vesting_amount: Uint128::new(5000u128),
                        cliff_amount: Uint128::new(1250u128),
                    },
                vesting_denom: cw20::Denom::Native("token".to_string()),
                vested_amount: Uint128::new(1250u128),
                claimable_amount: Uint128::new(1250u128),
            }]
        }
    );

    Ok(())
}
