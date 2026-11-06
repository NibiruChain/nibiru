# v2.23 Holder-Set Fixture

`usdc_holder_set.wasm.gz` is test data, not a runtime embed. The upgrade requires
the registry to have been deployed beforehand. Tests decompress these bytes,
upload and instantiate them through the real Wasm keeper, update the set, and
exercise the Go reader. No source-level contract mock substitutes for execution.

Decompressed SHA-256:

```text
fe453d5690f836f231b79b039b8561702a70d20cc5e34415a8faa6773037d4ca
```

Temporary source: `wasm-contracts/usdc-holder-set`. Build on the x86_64 homebox
from the pushed branch using `build-fixture.sh`, then copy back the generated
`.wasm.gz` and build provenance. The script pins compiler/lowering settings and
uses gzip timestamp zero. Update the fixture checksum after every rebuild.

This is a development fixture. The contract source may change before release,
so a final homebox rebuild and Keeper test pass are required even if the API
is unchanged. Select the final deployment checksum/address only afterward.

The current fixture was generated on homebox x86_64 from source revision
`20780e729`. File `build-info.json` records the
compiler, source revision, architecture, build flags, and both raw/gzip hashes.
This rebuild adds the chain-bound holding-address query and restricted admin
update. Opaque pagination cursors still allow reading beyond a malformed boundary
key. The complete upgrade, app, and report suites execute these exact bytes.
Build provenance identifies the source revision used for the Wasm, independently
of later Go and documentation commits.

Run the focused keeper tests:

```sh
go test ./app/upgrades -run 'TestUSDCHolder|TestUpgrade2_23_0' -count=1
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
| Deploy real fixture, apply final admin updates, read all candidates | Real Keeper tests on nibiru-localnet-0, nibiru-testnet-2, and cataclysm-1 |
| Same instantiate2 address despite different code IDs and initial contents | Real-bytecode Keeper parity tests |
| Registered v2.23 handler through evmtest.RunUpgrade | Full collection happy path on explicit localnet/testnet/mainnet chain IDs |
| Bank conversion, EOA and SaiEvm collection, protected contracts and Safe baseline | Exact balance and repayment/custody assertions in one run |
| Failed bank/EOA transfers and balance queries | Rollback, failed-account visibility, and continued later collection |
| Both bank and registry pagination at mainnet-like scale | 213 bank holders and 949 ERC20 holders in the complete collection fixture |
| Returned block events rather than only the transient test event manager | Scheduled BeginBlock execution plus event-only reconstruction |
| Repeated execution, source/readonly nonces, excess escrow | No duplicate collection, preserved nonces, reported donations |
| Fixture checksum and production registry profile remain aligned | Direct regression check against decompressed bytecode |

The handler runs required migrations and enabling transitions before custom
collection, following the earlier upgrades. The registry activates collection only
on the explicitly supported chains. Missing deployment skips collection; invalid
bytecode, configuration, pagination, mapping, or recipient checks report failure
before moving funds. Bank conversions retain an outer SDK cache. ERC20 transfers
use their StateDB snapshot and isolated events, committing only after postchecks.

The mainnet profile pins the legacy asset and already-replaced bank custody.
The testnet/localnet profile pins the disposable six-decimal ERC20-origin asset
`0x0F1fD9763cC431d7fceC1AFA925747619bbb7b96` and its deterministic recovery
fixture. Existing testnet Sai bank-origin collateral is not the rehearsal asset.

The authenticated Config query supplies the destination. Mainnet always uses
`0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2`; testnet/localnet default to
`0x447fC0471f5d8150A790e19000930108aCCe0BC7`. Only those two addresses are
permitted on rehearsal chains, and selecting the Safe requires deployed code.
Unknown chains do not collect. Registry presence alone cannot select an asset
or an arbitrary recipient.

With the pinned creator `nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l`, salt
`usdc-holders-v1`, and `fix_msg=false`, these bytes derive
`nibi1yrflth2rznwwj3u8h5autuq55hdgmlz6kshgarvnj7ylqup8cpnsefl0se`.
Deploy with that creator as execute admin and no Wasm migration admin. Further
Rust edits require another homebox build, checksum/address update, and Keeper
pass before deployment. Keeper tests do not perform cross-chain bridging or
broadcast repayments.

Use the read-only report helper after execution:

```sh
go run ./contrib/scripts/usdc-migration-report -input block-results.json > redistribution.json
```

It reconstructs from returned BeginBlock bank-conversion/EVM events and context
markers, verifies matching actual movements, and separates repayment from already
replaced custody. Missing or duplicate accounting is an error, not a partial plan.
