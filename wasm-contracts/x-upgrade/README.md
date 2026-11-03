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

The `history` query uses an exclusive `(height, name)` cursor, limit 1 to 200,
and optional descending order. Name lookup returns the earliest matching
record, preserving legacy behavior even if imported history contains duplicate
names. Module-version updates merge supplied entries rather than deleting
omitted modules.
