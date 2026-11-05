# v2.23 Holder-Set Fixture

`usdc_holder_set.wasm.gz` is test data, not a runtime embed. The upgrade requires
the registry to have been deployed beforehand. Tests decompress these bytes,
upload and instantiate them through the real Wasm keeper, update the set, and
exercise the Go reader. No source-level contract mock substitutes for execution.

Decompressed SHA-256:

```text
0a18618d6d1dcf9ef0cf4a92252abf879b96a22066b1e7e57e2efd1c958204c8
```

Temporary source: `wasm-contracts/usdc-holder-set`. Build on the x86_64 homebox
from the pushed branch using `build-fixture.sh`, then copy back the generated
`.wasm.gz` and build provenance. The script pins compiler/lowering settings and
uses gzip timestamp zero. Update the fixture checksum after every rebuild.

This is a development fixture. The contract source may change before release,
so a final homebox rebuild and Keeper test pass are required even if the API
is unchanged. Select the final deployment checksum/address only afterward.

The current fixture was generated on homebox x86_64 from source revision
`f6a5666cb765f6e339010e9a961d2305e4cdbeaa`. File `build-info.json` records the
compiler, source revision, architecture, build flags, and both raw/gzip hashes.
The exact copied bytes passed the focused Keeper and existing v2.22 tests.

Run the focused keeper tests:

```sh
go test ./app/upgrades -run TestUSDCHolderSet -count=1
```

The tests cover an admin's final update before reading, unauthorized and invalid
updates, more than 1,000 candidates across pages, wrong bytecode, missing
predeployment, and instantiate2 parity across localnet/testnet/mainnet despite
different code IDs and initial sets. After deployment and migration work is
complete, the temporary Rust source can be removed while this regression fixture
stays with the upgrade tests.

## Review Scope

| Behavior | Coverage |
| --- | --- |
| Deploy real fixture, apply admin updates, read all candidates | Keeper happy path on localnet, nibiru-testnet-3, and cataclysm-1 |
| Same instantiate2 address with different code IDs and initial contents | Real-bytecode Keeper test on all three chain IDs |
| Wrong checksum, missing predeployment, invalid or unauthorized updates | Covered by focused registry tests |
| Invoke the v2.23 handler through evmtest.RunUpgrade | Not implemented in this slice |
| Convert bank coins, collect EOA ERC20, collect SaiEvm and preserve other contracts | Not implemented in this slice |
| Reconstruct redistribution from upgrade-block events and prove rollback | Not implemented in this slice |

Current tests use `evmtest.NewTestDeps` and the Wasm Keeper directly. A passing
registry test is not evidence that a full upgrade or fund collection ran.

The next handler test should invoke `deps.RunUpgrade` with an explicit chain ID
and predeployed registry, then check bank conversion, EOA collection, controlled
contract collection, unrelated-contract exclusion, and event reconstruction.
It must distinguish required module migrations from best-effort custom steps,
following the existing v2.21 handler's cached-operation convention.

The proposed activation rule is a valid registry at the known shared address,
not a blanket mainnet-only condition. Missing deployment would skip the custom
USDC step with an explicit event; wrong bytecode or query failure would report a
custom-step failure before transfers. Required schema migrations still run.
This rule is a proposal for the collection handler, not code implemented here.
Token mappings and the Safe recipient also require validated per-chain setup;
registry presence alone does not establish those identities.
