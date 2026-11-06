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
`c244372a204a8e2f89fd510e14860226f5863b62`. File `build-info.json` records the
compiler, source revision, architecture, build flags, and both raw/gzip hashes.
The rebuilt bytes match the previous fixture exactly. The complete upgrade, app,
and report suites pass with this fixture on ARM64 and homebox x86_64.

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
| Deploy real fixture, apply final admin updates, read all candidates | Real Keeper tests on localnet, nibiru-testnet-3, and cataclysm-1 |
| Same instantiate2 address despite different code IDs and initial contents | Real-bytecode Keeper parity tests |
| Registered v2.23 handler through evmtest.RunUpgrade | Full collection happy path on explicit localnet/testnet/mainnet chain IDs |
| Bank conversion, EOA and SaiEvm collection, protected contracts and Safe baseline | Exact balance and repayment/custody assertions in one run |
| Failed bank/EOA transfers and balance queries | Rollback, failed-account visibility, and continued later collection |
| Both bank and registry pagination at mainnet-like scale | 213 bank holders and 949 ERC20 holders in the complete collection fixture |
| Returned block events rather than only the transient test event manager | Scheduled BeginBlock execution plus event-only reconstruction |
| Repeated execution, source/readonly nonces, excess escrow | No duplicate collection, preserved nonces, reported donations |
| Fixture checksum and production registry profile remain aligned | Direct regression check against decompressed bytecode |

The handler runs required migrations and enabling transitions before custom
collection, following the earlier upgrades. The registry is an activation
signal on any chain. Missing deployment skips the USDC step explicitly; wrong
bytecode, failed queries, or an invalid token/mapping/recipient profile report
custom-step failure without moving funds. Each holder operation is cached so
one failed collection does not undo successful collections or leak its own
partial bank/EVM writes and success logs.

The default profile contains reviewed mainnet identities. Other networks must
have matching, validated identities or receive a reviewed profile in the Go
handler before the shared binary is frozen. The registered handler selects only
the mainnet profile today; its fixture override is not runtime configuration.
The recorded testnet Sai collateral is bank-origin, so an ERC20-origin rehearsal
asset and verified Safe/custody identities are still needed. Deploying the
registry alone cannot prove collection there. The binary never guesses a token
or recipient from a name.

The source is temporary and the fixture remains provisional until the final
homebox build and predeployment verification. Local handler tests do not perform
the later cross-chain bridge round trip or broadcast repayments.

Use the read-only report helper after execution:

```sh
go run ./contrib/scripts/usdc-migration-report -input block-results.json > redistribution.json
```

It reconstructs from returned BeginBlock bank-conversion/EVM events and context
markers, verifies matching actual movements, and separates repayment from already
replaced custody. Missing or duplicate accounting is an error, not a partial plan.
