# Vendored Wasmer GitHub Actions (Nibiru)

This directory is an **upstream Wasmer CI copy**. GitHub does **not** run these
workflows from `nibi-wasm` — only repo-root [`.github/workflows/rust-test.yml`](../../../.github/workflows/rust-test.yml) runs vendored Wasmer validation via `just test-wasmer`.

## What Nibiru runs today

| Surface | Command | OS |
|---------|---------|-----|
| CosmWasm VM | `cargo test --all` (job `rust-test`) | ubuntu-24.04 x64 + arm64 |
| Vendored Wasmer | `just test-wasmer` (job `wasmer`) | ubuntu-latest only |

Shipped **libwasmvm** artifacts and CGO tests live in repo `wasm-go-wasmvm`, not here.

## Workflows kept (trimmed reference)

| File | Purpose |
|------|---------|
| [`workflows/test.yaml`](workflows/test.yaml) | Compiler/WAST stage matrix across OS targets (reference) |
| [`workflows/build.yml`](workflows/build.yml) | Upstream CLI/C-API release matrix (reference only; product crates removed) |

## Workflows removed (Phase 1.5)

Product-only or broken after vendored trim: `wasmer-config`, `wasmer-integration-tests`, `cloudcompiler`, `benchmark`, `documentation`, `cache-bucket-cleanup`, `check-public-api`.

## Cross-OS matrix notes

See epic `/epics/26-06-22-nibi-wasm-vendored-wasmer-trim.md` section **Cross-OS validation (deferred)** for when to port platform coverage from `test.yaml` into nibi-wasm CI.
