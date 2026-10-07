# Nibiru runtime source

This directory vendors Wasmer 7.4.2, commit
`7a48a071c7682a409d148cf37dc8da58322a123d`. The complete upstream source
archive has SHA256
`1959cc1a07ca96e583cac488832bfac3787c354edc69336d6ccbcaf2db49ed67`.

The workspace retains the native runtime, Singlepass, Cranelift, middleware,
and compiler/WAST test crates. Product SDK, WASIX, CLI, LLVM, examples, and
benchmarks are outside this workspace. The product artifact test module needs
upstream binary fixtures and is excluded; source serialization tests remain.
Compiler exclusions are copied from upstream 7.4.2 `tests/ignores.txt`.

Nibiru changes unsigned 64-bit ARM64 division's zero check to
`emit_cbz_label_far` in `lib/compiler-singlepass/src/machine_arm64.rs`.
Upstream already includes equivalent far branches for unsigned and signed
remainders. The v1 VM tests a large division function, zero-divisor traps,
and successful execution after a trap.

Wasmer 7 supplies its own compiler-builtins stack probes. The old OSR stack
management (including deferred popping and checked subtraction, also present
in upstream 5.0.6) has been replaced upstream; those old functions are not
copied into the new stack model.

[CWA-2026-006](https://github.com/CosmWasm/advisories/blob/main/CWAs/CWA-2026-006.md)
is public. CosmWasm's published runtime patch uses Wasmer 7.4.2. Its advisory
states Cosmos Labs' Singlepass license covers downstream usage. Preserve the
upstream Business Source License in `lib/compiler-singlepass/LICENSE`.

The source contains upstream static-memory protection changes
[`1e62ffb`](https://github.com/wasmerio/wasmer/commit/1e62ffbf298c6082ab889aac48d41fd86c5b7c78)
and
[`aa495eb`](https://github.com/wasmerio/wasmer/commit/aa495eb782ba80eb1a2e33893159146ddd12a2e1).
They enlarge the guard and enforce the static reservation. Their specific
relationship to the advisory is an inference from the patch, rather than an
explicit statement in the advisory.

`just rs test-wasmer` runs the retained library and compiler suites.
`just rs test cosmwasm-vm` checks the v1 integration, including memory traps,
gas limits, trap recovery, cache serialization, and the Sai perp fixture.
Chain builds use `contrib/scripts/build-wasmvm-source.sh` with Rust 1.95.0 and
locked dependencies. Its output includes a source digest, target, runtime/FFI
versions, and library checksum. Existing downloaded release archives are not
used by this path.

The runtime update changes consensus behavior and requires a coordinated
chain upgrade. Static PIE and the existing mainnet admission guard remain;
production upgrade height and admission changes require a separate release.
