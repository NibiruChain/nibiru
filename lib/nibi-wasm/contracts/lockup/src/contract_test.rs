//! Tests for the execute calls of contract.rs

use crate::contract::{execute, instantiate};
use crate::error::ContractError;
use crate::msgs::{ExecuteMsg, InstantiateMsg};
use crate::state::{locks, Lock, LockState, NOT_UNLOCKING_BLOCK_IDENTIFIER};

use cosmwasm_std::testing::{
    mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage,
};
use cosmwasm_std::{coins, BankMsg, Coin, Env, MessageInfo, OwnedDeps, SubMsg};

const OWNER: &str = "owner";
const USER: &str = "user";
const DENOM: &str = "unibi";

fn setup_contract() -> anyhow::Result<(
    OwnedDeps<MockStorage, MockApi, MockQuerier>,
    Env,
    MessageInfo,
)> {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let info = mock_info(OWNER, &[]);
    let msg = InstantiateMsg {};
    let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg)?;
    assert_eq!(0, res.messages.len());
    Ok((deps, env, info))
}

pub type TestResult = anyhow::Result<()>;

#[test]
fn test_execute_lock() -> TestResult {
    let (mut deps, env, _info) = setup_contract()?;

    // Successful lock
    let info = mock_info(USER, &coins(100, DENOM));
    let msg = ExecuteMsg::Lock { blocks: 100 };
    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(1, res.events.len());

    // Query the lock
    let locks = locks();
    let lock = locks.load(&deps.storage, 1)?;
    assert_eq!(lock.owner, USER);
    assert_eq!(lock.coin, Coin::new(100u128, DENOM));
    assert_eq!(lock.duration_blocks, 100);
    assert_eq!(lock.start_block, env.block.height);
    assert_eq!(lock.end_block, NOT_UNLOCKING_BLOCK_IDENTIFIER);
    assert!(!lock.funds_withdrawn);

    // Attempt to lock with no funds
    let info = mock_info(USER, &[]);
    let msg = ExecuteMsg::Lock { blocks: 100 };
    let err = execute(deps.as_mut(), env.clone(), info, msg).unwrap_err();
    assert!(matches!(err, ContractError::InvalidCoins(_)));

    // Attempt to lock with zero duration
    let info = mock_info(USER, &coins(100, DENOM));
    let msg = ExecuteMsg::Lock { blocks: 0 };
    let err = execute(deps.as_mut(), env, info, msg).unwrap_err();
    assert!(matches!(err, ContractError::InvalidLockDuration));
    Ok(())
}

#[test]
fn test_execute_initiate_unlock() -> TestResult {
    let (mut deps, env, _info) = setup_contract()?;

    // Create a lock first
    let info = mock_info(USER, &coins(100, DENOM));
    let msg = ExecuteMsg::Lock { blocks: 100 };
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    // Successful initiate unlock
    let msg = ExecuteMsg::InitiateUnlock { id: 1 };
    let info = mock_info(USER, &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(1, res.events.len());

    // Query the lock
    let locks = locks();
    let lock = locks.load(&deps.storage, 1)?;
    assert_eq!(lock.end_block, env.block.height + 100);

    // Attempt to initiate unlock again
    let msg = ExecuteMsg::InitiateUnlock { id: 1 };
    let info = mock_info(USER, &[]);
    let err = execute(deps.as_mut(), env.clone(), info, msg).unwrap_err();
    assert!(matches!(err, ContractError::AlreadyUnlocking(_)));

    // Attempt to initiate unlock for non-existent lock
    let msg = ExecuteMsg::InitiateUnlock { id: 99 };
    let info = mock_info(USER, &[]);
    let err = execute(deps.as_mut(), env, info, msg).unwrap_err();
    assert!(matches!(err, ContractError::NotFound(_)));
    Ok(())
}

#[test]
fn test_execute_withdraw_funds() -> TestResult {
    let (mut deps, mut env, _info) = setup_contract()?;

    // Create and initiate unlock for a lock
    let info = mock_info(USER, &coins(100, DENOM));
    let msg = ExecuteMsg::Lock { blocks: 100 };
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    let msg = ExecuteMsg::InitiateUnlock { id: 1 };
    let info = mock_info(USER, &[]);
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    // Attempt to withdraw before maturity
    let msg = ExecuteMsg::WithdrawFunds { id: 1 };
    let info = mock_info(USER, &[]);
    let err = execute(deps.as_mut(), env.clone(), info, msg).unwrap_err();
    assert!(matches!(err, ContractError::NotMatured(_)));

    // Fast forward to maturity
    env.block.height += 101;

    // Successful withdraw
    let msg = ExecuteMsg::WithdrawFunds { id: 1 };
    let info = mock_info(USER, &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg)?;
    assert_eq!(1, res.messages.len());
    assert_eq!(
        res.messages[0],
        SubMsg::new(BankMsg::Send {
            to_address: USER.to_string(),
            amount: vec![Coin::new(100u128, DENOM)]
        })
    );

    // Query the lock
    let locks = locks();
    let lock = locks.load(&deps.storage, 1)?;
    assert!(lock.funds_withdrawn);

    // Attempt to withdraw again
    let msg = ExecuteMsg::WithdrawFunds { id: 1 };
    let info = mock_info(USER, &[]);
    let err = execute(deps.as_mut(), env, info, msg).unwrap_err();
    assert!(matches!(err, ContractError::FundsAlreadyWithdrawn(_)));

    Ok(())
}

#[test]
fn test_lock_state() -> TestResult {
    let (mut _deps, env, _info) = setup_contract()?;

    let lock = Lock {
        id: 1,
        coin: Coin::new(100u128, DENOM),
        owner: USER.to_string(),
        duration_blocks: 100,
        start_block: env.block.height,
        end_block: NOT_UNLOCKING_BLOCK_IDENTIFIER,
        funds_withdrawn: false,
    };

    // Test FundedPreUnlock state
    assert_eq!(lock.state(env.block.height), LockState::FundedPreUnlock);

    // Test Unlocking state
    let mut unlocking_lock = lock.clone();
    unlocking_lock.end_block = env.block.height + 50;
    assert_eq!(unlocking_lock.state(env.block.height), LockState::Unlocking);

    // Test Matured state
    let mut matured_lock = lock.clone();
    matured_lock.end_block = env.block.height - 1;
    assert_eq!(matured_lock.state(env.block.height), LockState::Matured);

    // Test Withdrawn state
    let mut withdrawn_lock = lock;
    withdrawn_lock.funds_withdrawn = true;
    assert_eq!(withdrawn_lock.state(env.block.height), LockState::Withdrawn);

    Ok(())
}

#[test]
fn test_withdraw_at_exact_maturity() -> TestResult {
    let (mut deps, mut env, _info) = setup_contract()?;

    // Create and initiate unlock for a lock
    let info = mock_info(USER, &coins(100, DENOM));
    let msg = ExecuteMsg::Lock { blocks: 100 };
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    let msg = ExecuteMsg::InitiateUnlock { id: 1 };
    let info = mock_info(USER, &[]);
    let _ = execute(deps.as_mut(), env.clone(), info, msg)?;

    // Fast forward to exact maturity
    env.block.height += 100;

    // Attempt to withdraw at exact maturity
    let msg = ExecuteMsg::WithdrawFunds { id: 1 };
    let info = mock_info(USER, &[]);
    let res = execute(deps.as_mut(), env.clone(), info, msg)?;

    // Check that withdrawal was successful
    assert_eq!(1, res.messages.len());
    assert_eq!(
        res.messages[0],
        SubMsg::new(BankMsg::Send {
            to_address: USER.to_string(),
            amount: vec![Coin::new(100u128, DENOM)]
        })
    );

    // Query the lock to ensure it's marked as withdrawn
    let locks = locks();
    let lock = locks.load(&deps.storage, 1)?;
    assert!(lock.funds_withdrawn);

    Ok(())
}
