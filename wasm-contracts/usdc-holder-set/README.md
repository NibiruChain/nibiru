# Temporary USDC Holder Set

One-time deployment input for the v2.23 legacy-USDC migration. The contract
stores only EVM candidate addresses. The migration reads balances and contract
code from the chain at upgrade execution.

This is temporary source kept to build and review the deployment artifact.
After the registry is deployed and the migration no longer needs it, remove
this package and its Cargo lockfile entry. Keep the gzipped fixture and Keeper
regression tests under `app/upgrades/testdata/v2.23` and `app/upgrades`.
There is no native module, genesis hook, runtime embed, or automatic deployment.

## Interface

Instantiate with an explicit execute admin and up to 500 initial addresses:

```json
{"admin":"nibi1...","addresses":[]}
```

Only that admin can update the set. Additions/removals are limited to 500 entries
each per message, validated before writes, normalized to lowercase, and deduplicated.
An address cannot occur in both lists. Messages with attached funds are rejected.

```json
{"update_addresses":{"add":["0x0000000000000000000000000000000000000001"],"remove":[]}}
```

Public queries return ascending addresses with an exclusive cursor. The default
page is 200, and the accepted range is 1 through 500. A null next cursor marks
the final page, including when the final page is exactly full.

```json
{"addresses":{"start_after":null,"limit":200}}
```

```json
{"addresses":["0x0000000000000000000000000000000000000001"],"next_start_after":null}
```

The execute admin stored in contract state is separate from the optional Wasm
migration admin. No migrate entry point is provided. Deployment must explicitly
choose whether to set a migration admin; tests instantiate with none.

## Deterministic Deployment

Deploy before the upgrade on every target network. Use the same Wasm checksum,
actual instantiate-message creator and salt, with `fix_msg=false`. Code IDs,
chain IDs, execute admin, and initial holder contents may differ without changing
the predicted address. A multisig executing instantiate2 is the creator, not its
signer. The operator creator, deployment salt, and final shared address still
need to be selected and recorded before release wiring.

## Build and Test

From the repository root:

```sh
just rs test usdc-holder-set
```

The checked-in fixture was built natively using Rust 1.86.0 and release profile:

```sh
rustup target add --toolchain 1.86.0 wasm32-unknown-unknown
RUSTFLAGS='-C target-cpu=mvp' cargo +1.86.0 build --locked --release --target wasm32-unknown-unknown -p usdc-holder-set
bunx --package binaryen@116.0.0 wasm-opt \
  target/wasm32-unknown-unknown/release/usdc_holder_set.wasm \
  -o target/wasm32-unknown-unknown/release/usdc_holder_set.optimized.wasm \
  -Os --signext-lowering --mvp-features
sai-wasm-optimizer check target/wasm32-unknown-unknown/release/usdc_holder_set.optimized.wasm
```

The maintained optimizer's container build was unavailable during this slice
because the Docker daemon was not running. Its native `check` command validated
and compiled the fixture through Nibiru's maintained VM. Binaryen 116 lowers
the artifact to the instruction set accepted by the Keeper-linked runtime.
The real Keeper tests are required: standalone native validation accepted an
earlier build that the Keeper-linked runtime rejected. This is a native build,
not a claim that the pinned AMD64 optimizer-container release mode was used.

See `app/upgrades/testdata/v2.23/README.md` for the fixture checksum and gzip
reproduction command. Changing compiler settings or optimizing the bytes changes
the checksum and potentially the deployment address; choose the final artifact
before deploying the registry.
