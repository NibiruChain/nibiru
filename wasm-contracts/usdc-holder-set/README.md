# Temporary USDC Holder Set

One-time deployment input for the v2.23 legacy-USDC migration. The contract
stores EVM candidate addresses, its execute admin, and an optional rehearsal
holding address. The migration reads balances and contract code from the chain
at upgrade execution. The registry itself holds no migration transfer authority.

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

The public configuration query returns the execution chain, holding address, and
execute admin:

```json
{"config":{}}
```

```json
{"chain_id":"nibiru-testnet-2","holding_address":"0x447fC0471f5d8150A790e19000930108aCCe0BC7","admin":"nibi1..."}
```

Only the stored execute admin may change the rehearsal holding address:

```json
{"set_holding_address":{"holding_address":"0x447fC0471f5d8150A790e19000930108aCCe0BC7"}}
```

On `cataclysm-1`, Config always returns the immutable Safe
`0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2` and destination edits fail.
On `nibiru-testnet-2` and `nibiru-localnet-0`, Config defaults to op-b above.
The execute admin may select only op-b or that Safe. The chain comes from Env,
not a query argument. Unsupported chains reject Config and destination updates.
`UserAddr` accepts equivalent hexadecimal or Nibiru account forms by byte
identity. Go independently validates the destination and requires deployed code
when the Safe is selected.

The execute admin differs from the Wasm migration admin. Deploy with `ud-prod`
as execute admin and `--no-admin` for Wasm. There is no migrate entry point or
execute-admin rotation message. Every execute rejects attached funds.

## Deterministic Deployment

Deploy before the upgrade on every target network. Use the same Wasm checksum,
actual instantiate-message creator and salt, with `fix_msg=false`. Code IDs,
chain IDs, execute admin, and initial holder contents may differ without changing
the predicted address. A multisig executing instantiate2 is the creator, not its
signer. The selected creator and execute admin is `ud-prod`,
`nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l`, with salt `usdc-holders-v1`.
Current bytes derive `nibi1yrflth2rznwwj3u8h5autuq55hdgmlz6kshgarvnj7ylqup8cpnsefl0se`.
The testnet recovery fixture uses the distinct salt `usdc-recovery-fixture-v1`.
Verify the fixture checksum and on-chain metadata before using either instance.

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
update the decompressed checksum in the fixture test/reference and the v2.23
default deployment profile, and rerun the Keeper tests. The profile-alignment
regression test detects a rebuild that leaves the shipped checksum/address stale. No native optimizer CLI is required by the build script.

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
