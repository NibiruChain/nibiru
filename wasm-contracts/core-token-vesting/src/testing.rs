use crate::contract::tests::TestResult;
use crate::contract::{execute, instantiate, query};
use crate::errors::{CliffError, ContractError, VestingError};
use crate::msg::{
    Cw20HookMsg, ExecuteMsg, InstantiateMsg, QueryMsg, VestingAccountResponse,
    VestingData, VestingSchedule,
};

use cosmwasm_std::testing::{MockApi, MockQuerier, MockStorage};
use cosmwasm_std::MessageInfo;
use cosmwasm_std::{
    from_json,
    testing::{mock_dependencies, mock_env, mock_info},
    to_json_binary, Addr, Attribute, BankMsg, Coin, Env, OwnedDeps, Response,
    StdError, SubMsg, Timestamp, Uint128, Uint64, WasmMsg,
};
use cw20::{Cw20ExecuteMsg, Cw20ReceiveMsg, Denom};

#[test]
fn proper_initialization() -> TestResult {
    let mut deps = mock_dependencies();

    let msg = InstantiateMsg {};

    let info = mock_info("addr0000", &[]);

    let _res = instantiate(deps.as_mut(), mock_env(), info, msg)?;
    Ok(())
}

#[test]
fn register_cliff_vesting_account_with_native_token() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;

    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    let create_msg = |start_time: u64,
                      end_time: u64,
                      vesting_amount: u128,
                      cliff_amount: u128,
                      cliff_time: u64|
     -> ExecuteMsg {
        ExecuteMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVestingWithCliff {
                start_time: Uint64::new(start_time),
                end_time: Uint64::new(end_time),
                vesting_amount: Uint128::new(vesting_amount),
                cliff_amount: Uint128::new(cliff_amount),
                cliff_time: Uint64::new(cliff_time),
            },
        }
    };

    // zero amount vesting token
    let msg = create_msg(100, 110, 0, 1000, 105);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(0u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::ZeroVestingAmount),
    );

    // zero amount cliff token
    let msg = create_msg(100, 110, 1000, 0, 105);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(1000u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::Cliff(CliffError::ZeroAmount)),
    );

    // cliff time less than block time
    let msg = create_msg(100, 110, 1000, 1000, 99);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(1000u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::Cliff(CliffError::InvalidTime {
            cliff_time: 99,
            block_time: 100,
        })),
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
        ContractError::Vesting(
            CliffError::ExcessiveAmount {
                cliff_amount,
                vesting_amount,
            }
            .into(),
        ),
    );

    // deposit amount different than vesting amount
    let (vesting_amount, cliff_amount, cliff_time) = (1000, 250, 105);
    let msg = create_msg(100, 110, vesting_amount, cliff_amount, cliff_time);
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(999u128, "uusd")]),
        msg,
        ContractError::Vesting(
            VestingError::MismatchedVestingAndDepositAmount {
                vesting_amount: 1000u128,
                deposit_amount: 999u128,
            },
        ),
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
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;

    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // zero amount vesting token
    let msg = ExecuteMsg::RegisterVestingAccount {
        master_address: None,
        address: "addr0001".to_string(),
        vesting_schedule: VestingSchedule::LinearVesting {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            vesting_amount: Uint128::zero(),
        },
    };
    require_error(
        &mut deps,
        &env,
        mock_info("addr0000", &[Coin::new(0u128, "uusd")]),
        msg,
        ContractError::Vesting(VestingError::ZeroVestingAmount),
    );

    // normal amount vesting token
    let msg = ExecuteMsg::RegisterVestingAccount {
        master_address: None,
        address: "addr0001".to_string(),
        vesting_schedule: VestingSchedule::LinearVesting {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            vesting_amount: Uint128::new(1000000u128),
        },
    };

    // invalid amount
    let info = mock_info("addr0000", &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg.clone());
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => {
            assert_eq!(msg, "must deposit only one type of token")
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // invalid amount
    let info = mock_info(
        "addr0000",
        &[Coin::new(100u128, "uusd"), Coin::new(10u128, "ukrw")],
    );
    let res = execute(deps.as_mut(), env.clone(), info, msg.clone());
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => {
            assert_eq!(msg, "must deposit only one type of token")
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // invalid amount
    let info = mock_info("addr0000", &[Coin::new(10u128, "uusd")]);
    let res = execute(deps.as_mut(), env.clone(), info, msg.clone());
    match res {
        Err(err) => {
            assert_eq!(
                err,
                VestingError::MismatchedVestingAndDepositAmount {
                    vesting_amount: 1_000_000,
                    deposit_amount: 10
                }
                .into()
            )
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // valid amount
    let info = mock_info("addr0000", &[Coin::new(1000000u128, "uusd")]);
    let res: Response = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.attributes,
        vec![
            ("action", "register_vesting_account"),
            ("master_address", "",),
            ("address", "addr0001"),
            ("vesting_denom", "{\"native\":\"uusd\"}"),
            ("vesting_amount", "1000000"),
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
                master_address: None,
                vesting_denom: Denom::Native("uusd".to_string()),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::zero(),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::zero(),
            }],
        }
    );
    Ok(())
}

#[test]
fn register_vesting_account_with_cw20_token() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;
    let info = mock_info("token0000", &[]);
    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // zero amount vesting token
    let msg = ExecuteMsg::Receive(Cw20ReceiveMsg {
        sender: "addr0000".to_string(),
        amount: Uint128::new(1000000u128),
        msg: to_json_binary(&Cw20HookMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVesting {
                start_time: Uint64::new(100),
                end_time: Uint64::new(110),
                vesting_amount: Uint128::zero(),
            },
        })?,
    });

    // invalid zero amount
    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg);
    match res {
        Err(err) => {
            assert_eq!(
                err,
                ContractError::Vesting(VestingError::ZeroVestingAmount)
            )
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // invariant amount
    let msg = ExecuteMsg::Receive(Cw20ReceiveMsg {
        sender: "addr0000".to_string(),
        amount: Uint128::new(1000000u128),
        msg: to_json_binary(&Cw20HookMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVesting {
                start_time: Uint64::new(100),
                end_time: Uint64::new(110),
                vesting_amount: Uint128::new(999000u128),
            },
        })?,
    });

    // invalid amount
    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg);
    match res {
        Err(ContractError::Vesting(
            VestingError::MismatchedVestingAndDepositAmount {
                vesting_amount,
                deposit_amount,
            },
        )) => {
            assert_eq!(vesting_amount, 999000u128);
            assert_eq!(deposit_amount, 1000000u128);
        }
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // valid amount
    let msg = ExecuteMsg::Receive(Cw20ReceiveMsg {
        sender: "addr0000".to_string(),
        amount: Uint128::new(1000000u128),
        msg: to_json_binary(&Cw20HookMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVesting {
                start_time: Uint64::new(100),
                end_time: Uint64::new(110),
                vesting_amount: Uint128::new(1000000u128),
            },
        })?,
    });

    // valid amount
    let res: Response = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.attributes,
        vec![
            ("action", "register_vesting_account"),
            ("master_address", "",),
            ("address", "addr0001"),
            ("vesting_denom", "{\"cw20\":\"token0000\"}"),
            ("vesting_amount", "1000000"),
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
                master_address: None,
                vesting_denom: Denom::Cw20(Addr::unchecked("token0000")),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::zero(),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::zero(),
            }],
        }
    );
    Ok(())
}

#[test]
fn claim_native() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;

    // init env to time 100
    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // valid amount
    let msg = ExecuteMsg::RegisterVestingAccount {
        master_address: None,
        address: "addr0001".to_string(),
        vesting_schedule: VestingSchedule::LinearVesting {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            vesting_amount: Uint128::new(1000000u128),
        },
    };

    let info = mock_info("addr0000", &[Coin::new(1000000u128, "uusd")]);
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    // make time to half claimable
    env.block.time = Timestamp::from_seconds(105);

    // claim not found denom
    let msg = ExecuteMsg::Claim {
        denoms: vec![
            Denom::Native("ukrw".to_string()),
            Denom::Native("uusd".to_string()),
        ],
        recipient: None,
    };

    let info = mock_info("addr0001", &[]);
    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg);
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => assert_eq!(
            msg,
            "vesting entry is not found for denom {\"native\":\"ukrw\"}"
        ),
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // valid claim
    let msg = ExecuteMsg::Claim {
        denoms: vec![Denom::Native("uusd".to_string())],
        recipient: None,
    };

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
            Attribute::new("vesting_denom", "{\"native\":\"uusd\"}"),
            Attribute::new("vesting_amount", "1000000"),
            Attribute::new("vested_amount", "500000"),
            Attribute::new("claim_amount", "500000"),
        ],
    );

    // query vesting account
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
                master_address: None,
                vesting_denom: Denom::Native("uusd".to_string()),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::new(500000),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::zero(),
            }],
        }
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
            Attribute::new("vesting_denom", "{\"native\":\"uusd\"}"),
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
fn claim_cw20() -> TestResult {
    let mut deps = mock_dependencies();
    instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;

    // init env to time 100
    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // valid amount
    let msg = ExecuteMsg::Receive(Cw20ReceiveMsg {
        sender: "addr0000".to_string(),
        amount: Uint128::new(1000000u128),
        msg: to_json_binary(&Cw20HookMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVesting {
                start_time: Uint64::new(100),
                end_time: Uint64::new(110),
                vesting_amount: Uint128::new(1000000u128),
            },
        })?,
    });

    // valid amount
    let info = mock_info("token0001", &[]);
    execute(deps.as_mut(), env.clone(), info, msg)?;

    // make time to half claimable
    env.block.time = Timestamp::from_seconds(105);

    // claim not found denom
    let msg = ExecuteMsg::Claim {
        denoms: vec![
            Denom::Cw20(Addr::unchecked("token0002")),
            Denom::Cw20(Addr::unchecked("token0001")),
        ],
        recipient: None,
    };

    let info = mock_info("addr0001", &[]);
    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg);
    match res {
        Err(ContractError::Std(StdError::GenericErr { msg, .. })) => assert_eq!(
            msg,
            "vesting entry is not found for denom {\"cw20\":\"token0002\"}"
        ),
        _ => panic!("should not enter. got result: {res:?}"),
    }

    // valid claim
    let msg = ExecuteMsg::Claim {
        denoms: vec![Denom::Cw20(Addr::unchecked("token0001"))],
        recipient: None,
    };

    let res = execute(deps.as_mut(), env.clone(), info.clone(), msg.clone())?;
    assert_eq!(
        res.messages,
        vec![SubMsg::new(WasmMsg::Execute {
            contract_addr: "token0001".to_string(),
            funds: vec![],
            msg: to_json_binary(&Cw20ExecuteMsg::Transfer {
                recipient: "addr0001".to_string(),
                amount: Uint128::new(500000u128),
            })?,
        }),]
    );

    assert_eq!(
        res.attributes,
        vec![
            Attribute::new("action", "claim"),
            Attribute::new("address", "addr0001"),
            Attribute::new("vesting_denom", "{\"cw20\":\"token0001\"}"),
            Attribute::new("vesting_amount", "1000000"),
            Attribute::new("vested_amount", "500000"),
            Attribute::new("claim_amount", "500000"),
        ],
    );

    // query vesting account
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
                master_address: None,
                vesting_denom: Denom::Cw20(Addr::unchecked("token0001")),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::new(500000),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::zero(),
            }],
        }
    );

    // make time to half claimable
    env.block.time = Timestamp::from_seconds(110);

    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(
        res.messages,
        vec![SubMsg::new(WasmMsg::Execute {
            contract_addr: "token0001".to_string(),
            funds: vec![],
            msg: to_json_binary(&Cw20ExecuteMsg::Transfer {
                recipient: "addr0001".to_string(),
                amount: Uint128::new(500000u128),
            })?,
        }),]
    );
    assert_eq!(
        res.attributes,
        vec![
            Attribute::new("action", "claim"),
            Attribute::new("address", "addr0001"),
            Attribute::new("vesting_denom", "{\"cw20\":\"token0001\"}"),
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
fn query_vesting_account() -> TestResult {
    let mut deps = mock_dependencies();
    let _res = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info("addr0000", &[]),
        InstantiateMsg {},
    )?;

    // init env to time 100
    let mut env = mock_env();
    env.block.time = Timestamp::from_seconds(100);

    // native vesting
    let msg = ExecuteMsg::RegisterVestingAccount {
        master_address: None,
        address: "addr0001".to_string(),
        vesting_schedule: VestingSchedule::LinearVesting {
            start_time: Uint64::new(100),
            end_time: Uint64::new(110),
            vesting_amount: Uint128::new(1000000u128),
        },
    };

    let info = mock_info("addr0000", &[Coin::new(1000000u128, "uusd")]);
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    let msg = ExecuteMsg::Receive(Cw20ReceiveMsg {
        sender: "addr0000".to_string(),
        amount: Uint128::new(1000000u128),
        msg: to_json_binary(&Cw20HookMsg::RegisterVestingAccount {
            master_address: None,
            address: "addr0001".to_string(),
            vesting_schedule: VestingSchedule::LinearVesting {
                start_time: Uint64::new(100),
                end_time: Uint64::new(110),
                vesting_amount: Uint128::new(1000000u128),
            },
        })?,
    });

    // valid amount
    let info = mock_info("token0001", &[]);
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    // half claimable
    env.block.time = Timestamp::from_seconds(105);

    // query all entry
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
            vestings: vec![
                VestingData {
                    master_address: None,
                    vesting_denom: Denom::Cw20(Addr::unchecked("token0001")),
                    vesting_amount: Uint128::new(1000000),
                    vested_amount: Uint128::new(500000),
                    vesting_schedule: VestingSchedule::LinearVesting {
                        start_time: Uint64::new(100),
                        end_time: Uint64::new(110),
                        vesting_amount: Uint128::new(1000000u128),
                    },
                    claimable_amount: Uint128::new(500000),
                },
                VestingData {
                    master_address: None,
                    vesting_denom: Denom::Native("uusd".to_string()),
                    vesting_amount: Uint128::new(1000000),
                    vested_amount: Uint128::new(500000),
                    vesting_schedule: VestingSchedule::LinearVesting {
                        start_time: Uint64::new(100),
                        end_time: Uint64::new(110),
                        vesting_amount: Uint128::new(1000000u128),
                    },
                    claimable_amount: Uint128::new(500000),
                }
            ],
        }
    );

    // query one entry
    assert_eq!(
        from_json::<VestingAccountResponse>(&query(
            deps.as_ref(),
            env.clone(),
            QueryMsg::VestingAccount {
                address: "addr0001".to_string(),
                start_after: None,
                limit: Some(1),
            },
        )?)?,
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![VestingData {
                master_address: None,
                vesting_denom: Denom::Cw20(Addr::unchecked("token0001")),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::new(500000),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::new(500000),
            },],
        }
    );

    // query one entry after first one
    assert_eq!(
        from_json::<VestingAccountResponse>(&query(
            deps.as_ref(),
            env,
            QueryMsg::VestingAccount {
                address: "addr0001".to_string(),
                start_after: Some(Denom::Cw20(Addr::unchecked("token0001"))),
                limit: Some(1),
            },
        )?)?,
        VestingAccountResponse {
            address: "addr0001".to_string(),
            vestings: vec![VestingData {
                master_address: None,
                vesting_denom: Denom::Native("uusd".to_string()),
                vesting_amount: Uint128::new(1000000),
                vested_amount: Uint128::new(500000),
                vesting_schedule: VestingSchedule::LinearVesting {
                    start_time: Uint64::new(100),
                    end_time: Uint64::new(110),
                    vesting_amount: Uint128::new(1000000u128),
                },
                claimable_amount: Uint128::new(500000),
            }],
        }
    );
    Ok(())
}
