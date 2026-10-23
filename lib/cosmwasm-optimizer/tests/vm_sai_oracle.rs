//! Exercise a real Sai oracle artifact without a chain or native contract wrapper.
//! Set SAI_OPTIMIZER_ORACLE_WASM to the artifact and run this ignored test explicitly.

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
#[ignore = "requires compiled Sai oracle Wasm in SAI_OPTIMIZER_ORACLE_WASM"]
fn compiled_sai_oracle_updates_expiration_in_nibiru_vm() -> Result<()> {
    let path = PathBuf::from(
        env::var("SAI_OPTIMIZER_ORACLE_WASM")
            .context("set SAI_OPTIMIZER_ORACLE_WASM to compiled oracle.wasm")?,
    );
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

    let result = call_instantiate_raw(
        &mut instance,
        &env,
        &info,
        br#"{"owner":"creator","expiration_time":60}"#,
    )?;
    let response: ContractResult<Response<Empty>> =
        serde_json::from_slice(&result)?;
    assert!(response.is_ok(), "instantiate failed: {response:?}");

    let result = call_execute_raw(
        &mut instance,
        &env,
        &info,
        br#"{"update_expiration_time":{"expiration_time":120}}"#,
    )?;
    let response: ContractResult<Response<Empty>> =
        serde_json::from_slice(&result)?;
    assert!(response.is_ok(), "execute failed: {response:?}");

    let result =
        call_query_raw(&mut instance, &env, br#"{"expiration_time":{}}"#)?;
    let response: ContractResult<Binary> = serde_json::from_slice(&result)?;
    let data = response.into_result().map_err(anyhow::Error::msg)?;
    let expiration: u64 = serde_json::from_slice(data.as_slice())?;
    assert_eq!(expiration, 120);
    Ok(())
}
