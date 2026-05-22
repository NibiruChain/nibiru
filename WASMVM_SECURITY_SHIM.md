# WasmVM RUSTSEC-2024-0344 Shim

This is the narrow remediation path for `RUSTSEC-2024-0344` without moving
Nibiru off `wasmd v0.44.0`, Cosmos SDK 0.47, or the CosmWasm 1.5 runtime
interface.

## Current Dependency Path

Nibiru currently requires `github.com/CosmWasm/wasmd v0.44.0` and
`github.com/CosmWasm/wasmvm v1.5.9` in the root module. The internal wasmd fork
also requires `github.com/CosmWasm/wasmvm v1.5.9`.

Upstream `wasmvm v1.5.9` builds `libwasmvm` from CosmWasm `v1.5.11`. Its
`libwasmvm/Cargo.lock` resolves:

```text
cosmwasm-crypto 1.5.11
ed25519-zebra 3.1.0
curve25519-dalek 3.2.0
```

`curve25519-dalek 3.2.0` is the vulnerable package reported by
`RUSTSEC-2024-0344`.

## Fork And Tag

Create or use a Nibiru-controlled fork:

```text
repo: github.com/NibiruChain/wasmvm
base: github.com/CosmWasm/wasmvm tag v1.5.9
tag:  v1.5.9-nibiru.1
```

The tested fork diff is stored in
`contrib/patches/wasmvm-v1.5.9-nibiru.1.patch`. Apply it to a checkout of
upstream `github.com/CosmWasm/wasmvm` at tag `v1.5.9`:

```sh
git apply /path/to/nibiru/contrib/patches/wasmvm-v1.5.9-nibiru.1.patch
```

Keep the Go module path in the fork as `github.com/CosmWasm/wasmvm`. The Nibiru
module should consume it with a `replace`:

```go
require github.com/CosmWasm/wasmvm v1.5.9-nibiru.1

replace github.com/CosmWasm/wasmvm => github.com/NibiruChain/wasmvm v1.5.9-nibiru.1
```

Apply the same `require`/`replace` pair in `internal/wasmd/go.mod` so the
vendored wasmd fork can be tested as its own module.

## WasmVM Fork Patch

In the wasmvm fork, make these changes only:

1. Set `libwasmvm/Cargo.toml` package version to `1.5.9-nibiru.1`.
2. Vendor a patched copy of `cosmwasm-crypto 1.5.11` under
   `libwasmvm/vendor/cosmwasm-crypto-1.5.11-rustsec-2024-0344`.
3. In the vendored `cosmwasm-crypto` package, change only:

```toml
[dependencies.ed25519-zebra]
version = "=4.0.3"
default-features = false
```

4. Add this patch section to `libwasmvm/Cargo.toml`:

```toml
[patch."https://github.com/CosmWasm/cosmwasm.git"]
cosmwasm-crypto = { path = "vendor/cosmwasm-crypto-1.5.11-rustsec-2024-0344" }
```

5. Regenerate `libwasmvm/Cargo.lock` with wasmvm's production toolchain:

```sh
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo +1.73.0 update -p cosmwasm-crypto --manifest-path libwasmvm/Cargo.toml
```

The patched lockfile must contain `ed25519-zebra 4.0.3` and
`curve25519-dalek 4.1.3`, and must not contain `ed25519-zebra 3.1.0` or
`curve25519-dalek 3.2.0`.

## Release Artifacts

Nibiru release builds link prebuilt static `libwasmvm` archives. Publish these
assets on the Nibiru wasmvm release:

```text
https://github.com/NibiruChain/wasmvm/releases/download/v1.5.9-nibiru.1/libwasmvmstatic_darwin.a
https://github.com/NibiruChain/wasmvm/releases/download/v1.5.9-nibiru.1/libwasmvm_muslc.x86_64.a
https://github.com/NibiruChain/wasmvm/releases/download/v1.5.9-nibiru.1/libwasmvm_muslc.aarch64.a
```

Build them from the fork with wasmvm's existing release targets:

```sh
make release-build-alpine
make release-build-macos-static
```

The Nibiru build makefile derives the release repository and version from the
effective `github.com/CosmWasm/wasmvm` module. With the `replace` above, it
downloads from `NibiruChain/wasmvm` at `v1.5.9-nibiru.1`.

## Local Validation Completed

The patch shape was tested in a temporary wasmvm checkout from upstream
`v1.5.9` using Rust/Cargo `1.73.0`:

```text
cargo +1.73.0 update -p cosmwasm-crypto: succeeded
git apply contrib/patches/wasmvm-v1.5.9-nibiru.1.patch to fresh v1.5.9 checkout: succeeded
libwasmvm/Cargo.lock version: 3
cargo +1.73.0 build --release --example wasmvmstatic: succeeded
native static archive: target/release/examples/libwasmvmstatic.a
native static archive sha256: dda58f53f0f1e740fb4d6a2fba57c63f25908a782ad67e2d402e1f427f9f3687
embedded libwasmvm version string: 1.5.9-nibiru.1
```

`cargo tree -i curve25519-dalek` resolved:

```text
curve25519-dalek v4.1.3
ed25519-zebra v4.0.3
cosmwasm-crypto v1.5.11
cosmwasm-std v1.5.11
cosmwasm-vm v1.5.11
wasmvm v1.5.9-nibiru.1
```

The patched wasmvm checkout was also linked into Nibiru locally using temporary
module replacements:

```text
github.com/CosmWasm/wasmvm v1.5.9-nibiru.1 => /tmp/wasmvm-nibiru-173
```

With a pre-seeded local static archive at
`temp/wasmvm/1.5.9-nibiru.1/lib/darwin_all/libwasmvmstatic_darwin.a`:

```text
make build: succeeded
./build/nibid query wasm libwasmvm-version: 1.5.9-nibiru.1
go test -count=1 -tags "netgo osusergo ledger static rocksdb pebbledb static_wasm grocksdb_no_link" ./app/wasmext/...: succeeded
```

The patched wasmvm module's own Go tests also passed:

```text
CGO_ENABLED=1 go test -count=1 -tags static_wasm ./...
```

The temporary local module replacements were not kept in this repo because the
real deliverable requires a published Nibiru-controlled fork and release
artifacts.

## Nibiru Validation Checklist

After the fork tag and release assets exist:

```sh
go mod tidy
(cd internal/wasmd && go mod tidy)
go list -m github.com/CosmWasm/wasmvm
make build
go test ./app/wasmext/...
(cd internal/wasmd && go test ./x/wasm/...)
./build/nibid query wasm libwasmvm-version
```

Expected results:

- `go list -m github.com/CosmWasm/wasmvm` shows
  `github.com/NibiruChain/wasmvm v1.5.9-nibiru.1` as the replacement.
- `make build` downloads static artifacts from
  `github.com/NibiruChain/wasmvm/releases/download/v1.5.9-nibiru.1`.
- `nibid query wasm libwasmvm-version` reports `1.5.9-nibiru.1`.

## Removal

This is a temporary security shim. Remove the fork, `replace` directives, and
this document when Nibiru performs a full wasmd/CosmWasm upgrade that ships the
fixed crypto dependency directly.
