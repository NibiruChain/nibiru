# Add Wasm VM FFI CI and release automation

This PR adds the first Nibiru-native CI and release automation for the `lib/wasmvm-ffi` subtree, so the copied WasmVM FFI module can be tested, built, and prepared for artifact releases from the monorepo.

## Key Changes

1. Adds workflow `Wasm VM FFI` for the `lib/wasmvm-ffi` subtree, with path-filtered jobs for Rust base tests, Rust linting, Go base tests, Go linting, Go FFI tests, Docker artifact builds, and release publishing.

1. Routes workflow commands through command `just wasmvm ...`, keeping the workflow rooted at the Nibiru repository while delegating execution to the subtree `justfile`.

1. Adds command `just test-alpine` inside `lib/wasmvm-ffi` as the workflow-facing entrypoint for Alpine musl Go tests, while preserving the existing Makefile-backed implementation for now.

1. Updates the libwasmvm release helper to publish from repository `NibiruChain/nibiru` using Go submodule-style tags such as `lib/wasmvm-ffi/v1.6.0`, while preserving the existing `libwasmvm_*` release asset filenames.

## Manual Testing I Did Locally

- `bunx github-actionlint .github/workflows/wasmvm-ffi.yml`
- `just wasmvm ci-go`
- `just wasmvm release test`
- `just wasmvm release publish` dry run only
