use std::sync::Arc;
use wasmer::sys::BaseTunables;
#[cfg(feature = "cranelift")]
use wasmer::sys::Cranelift;
#[cfg(not(feature = "cranelift"))]
use wasmer::sys::Singlepass;
use wasmer::sys::{CompilerConfig, NativeEngineExt};
use wasmer::{wasmparser::Operator, Engine, Pages, WASM_PAGE_SIZE};

use crate::size::Size;

use super::gatekeeper::Gatekeeper;
use super::limiting_tunables::LimitingTunables;
use super::metering::{is_accounting, Metering};

/// WebAssembly linear memory objects have sizes measured in pages. Each page
/// is 65536 (2^16) bytes. In WebAssembly version 1, a linear memory can have at
/// most 65536 pages, for a total of 2^32 bytes (4 gibibytes).
/// https://github.com/WebAssembly/memory64/blob/master/proposals/memory64/Overview.md
const MAX_WASM_PAGES: u32 = 65536;

fn cost(operator: &Operator) -> u64 {
    // A flat fee for each operation
    // The target is 1 Teragas per millisecond (see GAS.md).
    //
    // In https://github.com/CosmWasm/cosmwasm/pull/1042 a profiler is developed to
    // identify runtime differences between different Wasm operation, but this is not yet
    // precise enough to derive insights from it.
    const GAS_PER_OPERATION: u64 = 115_000;

    if is_accounting(operator) {
        GAS_PER_OPERATION * 14
    } else {
        GAS_PER_OPERATION
    }
}

/// Creates an engine without a compiler.
/// This is used to run modules compiled before.
pub fn make_runtime_engine(memory_limit: Option<Size>) -> Engine {
    let mut engine = Engine::headless();
    if let Some(limit) = memory_limit {
        let base = BaseTunables::new();
        let tunables = LimitingTunables::new(base, limit_to_pages(limit));
        engine.set_tunables(tunables);
    }
    engine
}

/// Creates an Engine with a compiler attached. Use this when compiling Wasm to a module.
pub fn make_compiling_engine(memory_limit: Option<Size>) -> Engine {
    let gas_limit = 0;
    let deterministic = Arc::new(Gatekeeper::default());
    let metering = Arc::new(Metering::new(gas_limit, cost));

    #[cfg(feature = "cranelift")]
    let mut compiler = Cranelift::default();

    #[cfg(not(feature = "cranelift"))]
    let mut compiler = Singlepass::default();

    compiler.canonicalize_nans(true);
    compiler.push_middleware(deterministic);
    compiler.push_middleware(metering);
    let mut engine = Engine::from(compiler);
    if let Some(limit) = memory_limit {
        let base = BaseTunables::new();
        let tunables = LimitingTunables::new(base, limit_to_pages(limit));
        engine.set_tunables(tunables);
    }
    engine
}

fn limit_to_pages(limit: Size) -> Pages {
    // round down to ensure the limit is less than or equal to the config
    let limit_in_pages: usize = limit.0 / WASM_PAGE_SIZE;

    let capped = match u32::try_from(limit_in_pages) {
        Ok(x) => std::cmp::min(x, MAX_WASM_PAGES),
        // The only case where TryFromIntError can happen is when
        // limit_in_pages exceeds the u32 range. In this case it is way
        // larger than MAX_WASM_PAGES and needs to be capped.
        Err(_too_large) => MAX_WASM_PAGES,
    };
    Pages(capped)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn instantiate(wat: &str) -> (wasmer::Store, wasmer::Instance) {
        let mut store =
            wasmer::Store::new(make_compiling_engine(Some(Size::mebi(16))));
        let wasm = wat::parse_str(wat).unwrap();
        let module = wasmer::Module::new(&store, wasm).unwrap();
        let instance =
            wasmer::Instance::new(&mut store, &module, &wasmer::imports! {})
                .unwrap();
        instance
            .exports
            .get_global("wasmer_metering_remaining_points")
            .unwrap()
            .set(&mut store, wasmer::Value::I64(i64::MAX))
            .unwrap();
        (store, instance)
    }

    #[test]
    fn large_effective_memory_addresses_trap_and_instance_recovers() {
        // Wasm32 addresses plus unsigned static offsets can approach 8 GiB.
        // Exercise both reads and writes through the production engine/middleware.
        for offset in [0u32, 0x8000_0000, 0xffff_ffff] {
            let wat = format!(
                r#"(module (memory 1)
                (func (export "load") (param i32) (result i32)
                    local.get 0 i32.load offset={offset})
                (func (export "store") (param i32)
                    local.get 0 i32.const 17 i32.store offset={offset})
                (func (export "valid") (result i32)
                    i32.const 0 i32.const 42 i32.store
                    i32.const 0 i32.load))"#
            );
            let (mut store, instance) = instantiate(&wat);
            let load = instance
                .exports
                .get_typed_function::<i32, i32>(&store, "load")
                .unwrap();
            let write = instance
                .exports
                .get_typed_function::<i32, ()>(&store, "store")
                .unwrap();
            let valid = instance
                .exports
                .get_typed_function::<(), i32>(&store, "valid")
                .unwrap();
            for address in [0x7fff_ffffu32, 0xffff_ffff] {
                assert!(load
                    .call(&mut store, address as i32)
                    .unwrap_err()
                    .to_string()
                    .contains("out of bounds"));
                assert!(write
                    .call(&mut store, address as i32)
                    .unwrap_err()
                    .to_string()
                    .contains("out of bounds"));
                assert_eq!(valid.call(&mut store).unwrap(), 42);
            }
        }
    }

    #[test]
    fn large_integer_division_function_compiles_and_traps() {
        // This exceeds ARM64 CBZ range: restoring the short branch produces
        // ImpossibleRelocation during assembler finalization.
        let mut body =
            String::from("local.get 0 local.get 1 i64.div_u local.set 0 ");
        for _ in 0..100000 {
            body.push_str("local.get 0 i64.const 1 i64.add local.set 0 ");
        }
        body.push_str("local.get 0");
        let wat = format!("(module (func (export \"divide\") (param i64 i64) (result i64) {body}))");
        let (mut store, instance) = instantiate(&wat);
        let divide = instance
            .exports
            .get_typed_function::<(i64, i64), i64>(&store, "divide")
            .unwrap();
        assert_eq!(divide.call(&mut store, 100, 2).unwrap(), 100050);
        assert!(divide.call(&mut store, 100, 0).is_err());
        assert_eq!(divide.call(&mut store, 100, 2).unwrap(), 100050);
    }

    #[test]
    fn limit_to_pages_works() {
        // rounds down
        assert_eq!(limit_to_pages(Size(0)), Pages(0));
        assert_eq!(limit_to_pages(Size(1)), Pages(0));
        assert_eq!(limit_to_pages(Size::kibi(63)), Pages(0));
        assert_eq!(limit_to_pages(Size::kibi(64)), Pages(1));
        assert_eq!(limit_to_pages(Size::kibi(65)), Pages(1));
        assert_eq!(limit_to_pages(Size(u32::MAX as usize)), Pages(65535));
        // caps at 4 GiB
        assert_eq!(limit_to_pages(Size::gibi(3)), Pages(49152));
        assert_eq!(limit_to_pages(Size::gibi(4)), Pages(65536));
        assert_eq!(limit_to_pages(Size::gibi(5)), Pages(65536));
        assert_eq!(limit_to_pages(Size(usize::MAX)), Pages(65536));
    }
}
