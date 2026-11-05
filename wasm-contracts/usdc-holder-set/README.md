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

Build artifacts on the x86_64 homebox from the pushed development branch.
The build script refuses other architectures and source edits or additions that have
not been committed. It writes the Wasm, reproducible gzip, and build provenance
outside testdata so the checked-in fixture is replaced deliberately.

```sh
# On homebox, in the checkout of the pushed branch:
bash wasm-contracts/usdc-holder-set/build-fixture.sh
```

The script pins Rust 1.86.0, the Wasm MVP target, and Binaryen 116.0.0. Copy the
resulting `.wasm.gz` and `build-info.json` back into `app/upgrades/testdata/v2.23`,
update the decompressed checksum in the fixture test/reference, and rerun the
Keeper tests. No native optimizer CLI is required by the build script.

The first local development build exposed a compatibility difference: standalone
native VM validation accepted bytes rejected by the runtime linked into Keeper
tests. Binaryen MVP lowering produced compatible bytes. Treat real Keeper
execution of the homebox artifact as the acceptance check.

The bytecode is provisional while the contract is being edited. Rebuild on
homebox after the final source changes even if the query API stays unchanged.
Keep the deployment checksum and predicted instantiate2 address provisional
until those final bytes are selected.

See `app/upgrades/testdata/v2.23/README.md` for the fixture checksum and gzip
reproduction command. Changing compiler settings or optimizing the bytes changes
the checksum and potentially the deployment address; choose the final artifact
before deploying the registry.
