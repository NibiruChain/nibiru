# x-upgrade

The contract owns upgrade requests, completed-upgrade history, module versions,
and protocol version. Nibiru preserves the native upgrade message and query
interfaces through the Go module in `x/upgrade`.

See [the implementation specification](../../spec/08-contract-owned-upgrades.md)
for the authority model, enabling migration, and acceptance requirements.

Governance calls `schedule` and `cancel` through authenticated native adapters.
The current `x/sudo` root controls configuration, self-migration, and admin
changes. The contract stores no root owner. Native lifecycle operations use the
CosmWasm `sudo` entry point and trust the native host and root-controlled host
facilities.

A time request stores a target in consensus Unix nanoseconds. At the first
eligible block, selection fixes `height + activation_delay_blocks`. The default
is 1,000 blocks and the minimum is two. Changing policy afterward does not move
that fixed height. There is one pending request; another governance-approved
request replaces it.

## Build and Test

```sh
cargo test -p x-upgrade
just rs optimizer build --manifest-path Cargo.toml -p x-upgrade --output-dir wasm-contracts/artifacts --mode release
gzip -9 -n -c wasm-contracts/artifacts/x_upgrade.wasm > x/upgrade/x_upgrade.wasm.gz
```

Update `ContractWasmSHA256` in `x/upgrade/wasm.go` with the uncompressed artifact
hash. The embedded-code test verifies that hash. Keep the deterministic gzip in
the commit so node operators do not need a Rust toolchain.

## Addresses and Release Provenance

The Chain contract uses `Instantiate2` with a stable module creator and salt.
The Wasm checksum determines its initial address. Initialization uses
`fixMsg=false`, so different genesis values do not change that address.
The keeper method `PredictContractAddr` computes the initial address through
the bound Wasm bridge without querying chain state.

A fresh chain, including a localnet, uses the binary's embedded build. Changing
that build changes the predicted initial address without requiring a salt
change. After deployment, Wasm migrations replace the code at the existing
address. The keeper uses its stored address and the Wasm creator index to find
that deployment; it does not replace it with a prediction for a later build.
Genesis import likewise preserves the exported contract and address.

Before publishing the finalized v2.22 enabling release, preserve its original
artifact, checksum, creator, salt, and predicted address as release provenance.
Tests of that released deployment must use those frozen inputs. Tests of fresh
genesis must use the embedded build under test, so updating a localnet build
does not rewrite an assertion about mainnet's original address. If a later
binary needs both the historical bootstrap and a newer fresh-genesis artifact,
keep them versioned separately and make the historical handler select its
original artifact explicitly. Do not silently replace a released bootstrap
artifact or change its salt to accommodate a later contract version.

The `history` query uses an exclusive `(height, name)` cursor, limit 1 to 200,
and optional descending order. Name lookup returns the earliest matching
record, preserving legacy behavior even if imported history contains duplicate
names. Module-version updates merge supplied entries rather than deleting
omitted modules.
