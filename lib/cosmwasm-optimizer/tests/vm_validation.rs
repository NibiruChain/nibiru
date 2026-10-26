use std::fs;

use cosmwasm_vm::{capabilities_from_csv, internals::check_wasm};
use sai_wasm_optimizer::validation::{check_file, NIBIRU_CAPABILITIES};

#[test]
fn rejects_bulk_memory_instruction_during_static_validation() {
    let wasm = wat::parse_str(
        r#"(module
            (memory (export "memory") 1)
            (func (export "interface_version_8"))
            (func (export "allocate") (param i32) (result i32) i32.const 0)
            (func (export "deallocate") (param i32))
            (func (export "instantiate"))
            (func (export "execute")
                i32.const 0
                i32.const 0
                i32.const 0
                memory.copy)
        )"#,
    )
    .unwrap();
    // Static validation checks function bodies, so it rejects the opcode before
    // compilation reaches the VM gatekeeper.
    let static_error =
        check_wasm(&wasm, &capabilities_from_csv(NIBIRU_CAPABILITIES))
            .unwrap_err();
    assert!(
        static_error
            .to_string()
            .to_ascii_lowercase()
            .contains("bulk memory"),
        "expected static validation rejection, got {static_error}"
    );
    let directory = tempfile::tempdir().unwrap();
    let artifact = directory.path().join("bulk-memory.wasm");
    fs::write(&artifact, wasm).unwrap();
    let error = check_file(&artifact).unwrap_err();
    assert!(
        format!("{error:#}")
            .to_ascii_lowercase()
            .contains("bulk memory"),
        "expected optimizer rejection, got {error:#}"
    );
}
