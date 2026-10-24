use std::{fs, path::Path};

use anyhow::{Context, Result};
use cosmwasm_vm::{
    capabilities_from_csv,
    internals::{check_wasm, compile, make_compiling_engine},
};

/// Capabilities enabled by Nibiru's Wasm keeper in `app/keepers.go`.
pub const NIBIRU_CAPABILITIES: &str =
    "iterator,staking,stargate,cosmwasm_1_1,cosmwasm_1_2,cosmwasm_1_3,cosmwasm_1_4";

/// Check static compatibility and compile with Nibiru's maintained VM.
/// Execution tests are still required to establish contract behavior.
pub fn check_file(path: &Path) -> Result<()> {
    let wasm = fs::read(path)
        .with_context(|| format!("reading Wasm artifact {}", path.display()))?;
    check_wasm(&wasm, &capabilities_from_csv(NIBIRU_CAPABILITIES))
        .with_context(|| format!("Nibiru VM rejected {}", path.display()))?;
    // Static checks defer function-body validation. Compilation also enforces
    // the VM gatekeeper's opcode policy, as the chain's store_code path does.
    compile(&make_compiling_engine(None), &wasm).with_context(|| {
        format!("Nibiru VM could not compile {}", path.display())
    })?;
    Ok(())
}
