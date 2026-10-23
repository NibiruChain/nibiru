# sai-wasm-optimizer

Build CosmWasm contracts with pinned Rust and Binaryen tools, then validate
artifacts with Nibiru's maintained VM. The CLI runs ARM64 tools on ARM64 Docker
and AMD64 tools on AMD64 Docker. Release mode always selects AMD64.

## Install

Build from the Nibiru repository with its pinned Rust toolchain:

```sh
cargo install --locked --path lib/cosmwasm-optimizer
sai-wasm-optimizer --help
sai-wasm-optimizer doctor
```

Docker must be running and accessible to the current user. The first build
pulls an optimizer image and downloads contract dependencies. The CLI itself
links Nibiru's maintained VM, so its initial compilation also builds Wasmer.
Install the normal native Rust build prerequisites for your platform.

For development in this checkout, use `just rs optimizer` instead of installing
the binary. For example, `just rs optimizer doctor`.

## Build and validate

```sh
# Build one contract from a workspace.
sai-wasm-optimizer build --manifest-path /path/to/sai-perps/Cargo.toml \
  --package oracle --output-dir /path/to/outputs/oracle

# Build every contract using native container tools.
sai-wasm-optimizer build --manifest-path /path/to/sai-perps/Cargo.toml

# Inspect selected packages and commands without compiling contracts.
sai-wasm-optimizer build --manifest-path Cargo.toml --dry-run

# Build AMD64 release artifacts, including on ARM hosts with emulation.
sai-wasm-optimizer build --manifest-path Cargo.toml --mode release

# Validate existing artifact files.
sai-wasm-optimizer check artifacts/*.wasm
```

From Nibiru's root, `just rs wasm-all` builds contracts under `wasm-contracts/`
and writes `wasm-contracts/artifacts/`. Command `just rs wasm-all release`
selects AMD64 release tools. Command `just rs wasm-check` validates those files.

Package discovery uses Cargo metadata and selects workspace members with a
`cdylib` target. Repeat `--package` to select more than one contract. Each
package builds separately to avoid merging Cargo features across selected
contracts. The CLI honors upstream `[package.metadata.optimizer]` standard
builds and named feature variants. It supports both Sai's `contracts/` tree and
Nibiru's `wasm-contracts/` tree. A shared helper that declares a `cdylib`
can opt out with `[package.metadata.optimizer]` and `standard-build = false`,
with no named builds. Sai uses this for its `packages/utils` library.

Output paths are relative to the caller's working directory. Use a dedicated
output directory for each build selection. A successful build replaces that
directory's Wasm files with the selected artifacts. Compilation or validation
failure leaves the previous artifacts in place. Generated Wasm files belong
in local build output or release assets, not source control.

## Tool versions and caches

The first implementation pins `cosmwasm/optimizer:0.17.0` for AMD64 and
`cosmwasm/optimizer-arm64:0.17.0` for ARM64, each by immutable image digest.
Both supply Rust 1.86.0 and Binaryen 116. The CLI selects that compiler
explicitly; a mounted repository's `rust-toolchain.toml` cannot replace it.
The compiler used to build the CLI can differ from the contract compiler.

Docker target and registry volumes are separated by workspace, architecture,
and toolchain. Compiler output lives at `/target`, matching optimizer 0.17.0.
The source workspace mounts read-only. The CLI copies artifact bytes back to
files owned by the calling user, so builds do not require `sudo chown`.

Every selected artifact must pass maintained-VM validation before the CLI
publishes the result. Output includes SHA-256 checksums and build metadata.
Compatibility checks cover imports, exports, capabilities, and Wasm features.
They do not establish contract behavior. The compiled-Wasm integration test
also instantiates, executes, and queries the counter fixture.

## WSL and Apple Silicon

On ARM64 WSL, run Docker's Linux ARM64 daemon and use the default development
mode. Apple Silicon users can use the same Linux ARM64 container path through
Docker Desktop. Apple Silicon hardware verification is tracked separately.
The CLI chooses the Docker daemon's architecture, which can differ from the
computer running the CLI when using a remote Docker context. Bind mounts must
refer to paths available to that daemon.

Release mode uses AMD64 even on ARM64. It requires working AMD64 emulation or
an AMD64 Docker daemon. An `exec format error` usually means the selected
architecture cannot execute in that Docker environment. Run `doctor` and
`doctor --mode release` to inspect the distinction.

ARM64 and AMD64 builds may produce different hashes. V1 supports ARM64 for
local development and retains AMD64 for releases. It does not claim identical
output across architectures. Source verification must use the recorded tool
versions, architecture, and image digest.

## Tests

```sh
just rs optimizer-test
just rs optimizer-vm-test /absolute/path/to/simple_contract.wasm
just rs optimizer-oracle-test /absolute/path/to/sai-perps/artifacts/oracle.wasm
```

The counter command uses the unchanged fixture from
[CosmWasm optimizer v0.17.0](https://github.com/CosmWasm/optimizer/tree/v0.17.0/tests/simple-contract).
It exercises emitted Wasm in Nibiru's maintained runtime. The oracle command
instantiates a real Sai oracle, changes its expiration, and queries the result.
The fixture binary
is an external build artifact and is not committed.

## Running without Docker

Docker packages the host Rust compiler, Wasm standard library, linker, and
Binaryen executable. A native backend needs each component at a selected
version, plus the same Cargo flags and VM checks. Moving orchestration into a
Rust binary does not supply these dependencies automatically.

The initial ARM64 experiment runs Binaryen 116 directly on WSL from the pinned
image's statically linked executable. Native compiler and artifact comparison
results are recorded in the child epic. A supported installer and native
backend remain follow-up work; V1 uses Docker for contract compilation.

## Upstream maintenance

The baseline is [CosmWasm/optimizer v0.17.0](https://github.com/CosmWasm/optimizer/tree/v0.17.0),
commit `b09e2fcf849b98ea846d6839f728bd4568dcd1fd`. Upstream optimizer is
Apache-2.0 licensed. This CLI invokes its published tools and preserves its
Cargo stripping flag and Binaryen `-Os` optimization recipe. Container
components retain their own licenses.

When updating the pins, rebuild the counter and Sai contracts on both
architectures, run maintained-VM execution tests, compare checksums, and record
any release-policy change. Keep the contract compiler pin and image digests
in sync.
