# Rust contract workspace

- Status: active
- Code scope: root `Cargo.toml`, root `just-rs.just`, directory `wasm-contracts/`,
  and Rust crates under `lib/`
- Source of truth: root `Cargo.toml` and file `just-rs.just`

Nibiru's Rust contract workspace lives at the repository root. Smart contracts
live under directory `wasm-contracts/`, and reusable crates live under
directory `lib/`.

## Workspace map

| Location | Contents |
| --- | --- |
| `wasm-contracts/` | Nibiru CosmWasm contracts, including `nibi-stargate`, incentives, lockup, multisig, vesting, and test contracts. |
| `lib/nibiru-std/` | Nibiru-specific types and Stargate bindings for CosmWasm contracts. |
| `lib/nibiru-ownable/` | Two-step ownership and delegated-permission patterns for contracts. |
| `lib/nibiru-ownable-derive/` | Procedural macros re-exported by crate `nibiru-ownable`. |
| `lib/cw-address-like/` | Address-like helper traits for CosmWasm types. |
| `lib/easy-addr/` | Address constructors used by tests. |
| `contrib/scripts/` | Rust helper programs and release scripts. |

Workspace file `Cargo.toml` excludes Wasmer and the Wasm VM FFI crate because
they keep independent Cargo workspaces and lockfiles.

## Commands

Use the root Rust dispatcher. Command `just rs setup` lists the available Rust
recipes.

```bash
just rs test-all
just rs build
just rs fmt-check
just rs clippy-check
just rs wasm-all
just rs wasm-check
```

Command `just rs test-all` runs the root Cargo workspace tests. Command
`just rs wasm-all` compiles contract artifacts, and command `just rs wasm-check`
validates those artifact files for deployment.

Each contract owns its usage documentation in its package `README.md`. Start
with [the contracts README](../wasm-contracts/README.md) when navigating the
contract tree.
