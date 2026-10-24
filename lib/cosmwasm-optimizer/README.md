# sai-wasm-optimizer

Build CosmWasm contracts with pinned Rust and Binaryen tools, then validate
artifacts with Nibiru's maintained VM. The CLI runs ARM64 tools on ARM64 Docker
and AMD64 tools on AMD64 Docker. Release mode always selects AMD64.

## 1 - Install

Native binaries are distributed through optimizer-specific GitHub releases.
The commands below require a published optimizer release; source installation
remains available while the first release is being prepared.

Install [cargo-binstall](https://github.com/cargo-bins/cargo-binstall#installation),
then download the manifest for the version you want and install through Cargo.
These commands work outside a Nibiru checkout:

```sh
manifest_dir="$(mktemp -d)"
curl --proto '=https' --tlsv1.2 --fail --location \
  https://github.com/NibiruChain/nibiru/releases/download/sai-wasm-optimizer-v0.1.0/sai-wasm-optimizer.toml \
  --output "$manifest_dir/sai-wasm-optimizer.toml"
cargo binstall --manifest-path "$manifest_dir/sai-wasm-optimizer.toml" \
  --strategies crate-meta-data --no-confirm sai-wasm-optimizer
sai-wasm-optimizer --version
sai-wasm-optimizer doctor
rm -r "$manifest_dir"
```

Binstall selects the native executable automatically. The release-metadata
strategy downloads the binary from the versioned release. Missing or unsupported
binaries fail rather than compiling from source or using a third-party binary.
Installation does not require a Rust compiler or publishing VM crates to
crates.io. If Cargo itself is unavailable, invoke `cargo-binstall` directly with
the same arguments. The executable stays installed for subsequent commands;
repeated installation of the same version reuses the installed binary.

A release also includes a `SHA256SUMS` file for verifying its downloads. From a
Nibiru checkout, the optional Just wrapper downloads the manifest, checks its
SHA-256 and requested version, then invokes `cargo-binstall` with the same
release-metadata strategy:

```sh
just rs optimizer-install 0.1.0
```

The wrapper requires `cargo-binstall`, `curl`, and `sha256sum` or `shasum`.
For an isolated installation, its second argument selects a Cargo installation
root. Direct Binstall users can add `--root /tmp/sai-optimizer` to the command
above. The wrapper equivalent is:

```sh
just rs optimizer-install 0.1.0 /tmp/sai-optimizer
/tmp/sai-optimizer/bin/sai-wasm-optimizer --help
```

The supported release targets are:

| Platform | Target | Runtime baseline |
| --- | --- | --- |
| Linux AMD64 | x86_64-unknown-linux-gnu | glibc 2.35, Ubuntu 22.04 |
| Linux ARM64 | aarch64-unknown-linux-gnu | glibc 2.39, Ubuntu 24.04 |
| macOS Intel | x86_64-apple-darwin | macOS 13 |
| macOS Apple Silicon | aarch64-apple-darwin | macOS 13 |

ARM64 WSL uses the Linux ARM64 binary. Alpine Linux and native Windows do not
have release targets. Mac builds validate a compiled Wasm fixture; Docker
Desktop contract-building behavior still needs hardware verification.

Binstall's ordinary name-only command discovers crates.io metadata. This
monorepo instead ships a dependency-free manifest as a release asset, with the
same version and download layout as the executable archives. Binstall accepts
that file through its `--manifest-path` option. Its `--git` option discovers
workspace members but cannot pin a Git tag or revision, so the installer uses
versioned release metadata. The original source crate remains in
`lib/cosmwasm-optimizer`.

To compile from source with Nibiru's pinned Rust toolchain:

```sh
cargo install --locked --path lib/cosmwasm-optimizer
```

Source compilation requires native Rust build prerequisites and compiles the
linked Nibiru VM and Wasmer. For checkout development, use the command
`just rs optimizer doctor` instead of installing the binary.

Docker must be running and accessible for contract builds. Native binary
installation removes host-tool compilation; it does not remove Docker or
contract compilation. The first contract build pulls its pinned optimizer
image and downloads contract dependencies.

## 2 - Build and validate

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

## 3 - Tool versions and caches

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

## 4 - WSL and Apple Silicon

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

## 5 - Tests

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

## 6 - Running without Docker

Docker packages the contract Rust compiler, Wasm standard library, linker, and
Binaryen executable. A native backend needs each component at a selected
version, plus the same Cargo flags and VM checks. Moving orchestration into a
Rust binary does not supply these dependencies automatically.

The initial ARM64 experiment runs Binaryen 116 directly on WSL from the pinned
image's statically linked executable. Native compiler and artifact comparison
results are recorded in the child epic. A native contract compiler backend remains follow-up work; the binary installer
distributes the host CLI, while V1 uses Docker for contract compilation.

## 7 - Upstream maintenance

The baseline is [CosmWasm/optimizer v0.17.0](https://github.com/CosmWasm/optimizer/tree/v0.17.0),
commit `b09e2fcf849b98ea846d6839f728bd4568dcd1fd`. Upstream optimizer is
Apache-2.0 licensed. This CLI invokes its published tools and preserves its
Cargo stripping flag and Binaryen `-Os` optimization recipe. Container
components retain their own licenses.

When updating the pins, rebuild the counter and Sai contracts on both
architectures, run maintained-VM execution tests, compare checksums, and record
any release-policy change. Keep the contract compiler pin and image digests
in sync.


## 8 - Package and release the CLI

Use Python 3.11 or later and Rust 1.95.0 on a native target runner:

```sh
just rs optimizer-release version
just rs optimizer-release validate-tag --tag sai-wasm-optimizer-v0.1.0
just rs optimizer-release package --target aarch64-unknown-linux-gnu --output-dir dist/optimizer
```

The package helper builds only the optimizer executable and its dependencies.
It overrides workspace LTO with 16 codegen units for host-tool build speed;
contract compiler flags and image pins are separate. Each archive contains the
executable, licenses, README, and JSON provenance. Packaging extracts and runs
the archived executable, checks its version, and validates the committed
`hackatom_1.0.wasm` fixture through Nibiru's VM without Docker.

The workflow in `.github/workflows/optimizer-release.yml` runs only when you
manually dispatch it or push an optimizer version tag. It has two job
definitions: native packaging with a four-platform matrix, and publication
after all packages pass. Manual runs upload archives and SHA-256 sidecars for
review. Tag runs additionally publish the release. Native Linux builders
establish the glibc baselines in the table; Mac builds target macOS 13.

### 8.1 - Build artifacts on demand

After the workflow is merged into the default branch, select **Optimizer native
distribution** in GitHub Actions, choose **Run workflow**, and select the branch
or tag to build. The equivalent CLI command is:

```sh
gh workflow run optimizer-release.yml --repo NibiruChain/nibiru --ref main
```

Download the four `optimizer-dist-*` artifacts from that run and collect their
archives and checksum sidecars in one directory. To inspect the bundle from a
checkout of the exact commit that produced it:

```sh
just rs optimizer-release publish --artifacts-dir dist/optimizer --tag sai-wasm-optimizer-v0.1.0
```

This command verifies the complete platform set, archive checksums, clean source
provenance, source commit, and lockfile hash. It generates the standalone
`sai-wasm-optimizer.toml` manifest and `SHA256SUMS` file and prints the publication
command. Only adding `--run` creates a GitHub release.

### 8.2 - Release successive versions

1. Set the version in `lib/cosmwasm-optimizer/Cargo.toml`. Use `0.1.0` for the
   first release and a new version for each later release. Run
   `cargo check --package sai-wasm-optimizer` to update the root lockfile when
   the package version changes, then commit both files through the normal PR
   process.
2. Choose the reviewed commit to release. Optionally run the manual build above
   on that commit's branch or tag and inspect its four artifacts.
3. Tag that exact commit with the matching version. For example, after reviewing
   the `0.1.0` source on main:

   ```sh
   git fetch origin main
   git tag sai-wasm-optimizer-v0.1.0 origin/main
   git push origin sai-wasm-optimizer-v0.1.0
   ```

4. Watch the **Optimizer native distribution** tag run. Each native runner
   validates that the tag matches the package version before building. Once
   all four packages pass, the publication job verifies and uploads the archives,
   standalone manifest, and checksum list.
5. Install the released version with `cargo binstall` using its downloaded
   manifest. Verify the executable's version and run `doctor`. Consuming
   repositories such as Sai can then pin that release manifest and install the
   binary during setup.

Optimizer releases use `--latest=false` so they do not replace the latest chain
release. Existing chain and libwasmvm release tag patterns are separate.
Published versions are immutable; use a new version and tag for corrections.
