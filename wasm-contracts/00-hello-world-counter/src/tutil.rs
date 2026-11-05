//! tutil.rs: Test helpers for the contract
#![cfg(not(target_arch = "wasm32"))]

use cosmwasm_std::{Env, MessageInfo, OwnedDeps};

#[cfg(not(target_arch = "wasm32"))]
use cosmwasm_std::testing::{
    mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage,
};

use crate::{contract::instantiate, msg::InstantiateMsg};

pub const TEST_OWNER: &str = easy_addr::addr!("owner");

pub fn setup_contract(
    count: i64,
) -> anyhow::Result<(
    OwnedDeps<MockStorage, MockApi, MockQuerier>,
    Env,
    MessageInfo,
)> {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let info = mock_info(TEST_OWNER, &[]);
    let msg = InstantiateMsg { count };
    let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg)?;
    assert_eq!(0, res.messages.len());
    Ok((deps, env, info))
}

pub fn setup_contract_defaults() -> anyhow::Result<(
    OwnedDeps<MockStorage, MockApi, MockQuerier>,
    Env,
    MessageInfo,
)> {
    setup_contract(0)
}

pub fn mock_info_for_sender(sender: &str) -> MessageInfo {
    mock_info(sender, &[])
}

pub fn mock_env_height(height: u64) -> Env {
    let mut env = mock_env();
    env.block.height = height;
    env
}
