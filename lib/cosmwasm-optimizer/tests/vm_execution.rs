//! Run against a compiled optimizer `tests/simple-contract` fixture:
//! `SAI_OPTIMIZER_TEST_WASM=/absolute/path/simple_contract.wasm cargo test
//! -p sai-wasm-optimizer --test vm_execution -- --ignored`
//! The fixture comes from CosmWasm/optimizer v0.17.0. No Wasm binary is committed.

use std::{env, fs, path::PathBuf};

use anyhow::{Context, Result};
use cosmwasm_std::{Binary, ContractResult, Empty, Response};
use cosmwasm_vm::{
    call_execute_raw, call_instantiate_raw, call_query_raw,
    capabilities_from_csv,
    testing::{
        mock_env, mock_info, mock_instance_with_options, MockInstanceOptions,
    },
};
use sai_wasm_optimizer::validation::{check_file, NIBIRU_CAPABILITIES};

#[test]
#[ignore = "requires compiled optimizer counter Wasm in SAI_OPTIMIZER_TEST_WASM"]
fn compiled_counter_executes_in_nibiru_vm() -> Result<()> {
    let path = PathBuf::from(env::var("SAI_OPTIMIZER_TEST_WASM").context(
        "set SAI_OPTIMIZER_TEST_WASM to compiled simple_contract.wasm",
    )?);
    check_file(&path)?;
    let wasm = fs::read(&path)?;
    let mut instance = mock_instance_with_options(
        &wasm,
        MockInstanceOptions {
            available_capabilities: capabilities_from_csv(NIBIRU_CAPABILITIES),
            ..Default::default()
        },
    );
    let env = serde_json::to_vec(&mock_env())?;
    let info = serde_json::to_vec(&mock_info("creator", &[]))?;

    let result =
        call_instantiate_raw(&mut instance, &env, &info, br#"{"count":17}"#)?;
    let response: ContractResult<Response<Empty>> =
        serde_json::from_slice(&result)?;
    assert!(response.is_ok(), "instantiate failed: {response:?}");

    let result =
        call_execute_raw(&mut instance, &env, &info, br#"{"increment":{}}"#)?;
    let response: ContractResult<Response<Empty>> =
        serde_json::from_slice(&result)?;
    assert!(response.is_ok(), "execute failed: {response:?}");

    let result = call_query_raw(&mut instance, &env, br#"{"get_count":{}}"#)?;
    let response: ContractResult<Binary> = serde_json::from_slice(&result)?;
    let data = response.into_result().map_err(anyhow::Error::msg)?;
    let count: serde_json::Value = serde_json::from_slice(data.as_slice())?;
    assert_eq!(count, serde_json::json!({"count":18}));
    Ok(())
}
