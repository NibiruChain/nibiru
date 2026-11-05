# v2.23 Holder-Set Fixture

`usdc_holder_set.wasm.gz` is test data, not a runtime embed. The upgrade requires
the registry to have been deployed beforehand. Tests decompress these bytes,
upload and instantiate them through the real Wasm keeper, update the set, and
exercise the Go reader. No source-level contract mock substitutes for execution.

Decompressed SHA-256:

```text
ae3840f748807eb93400ea7d0208ae084c002148ed60177fbf276a1ee407003f
```

Temporary source: `wasm-contracts/usdc-holder-set`. Build on the x86_64 homebox
from the pushed branch using `build-fixture.sh`, then copy back the generated
`.wasm.gz` and build provenance. The script pins compiler/lowering settings and
uses gzip timestamp zero. Update the fixture checksum after every rebuild.

This is a development fixture. The contract source may change before release,
so a final homebox rebuild and Keeper test pass are required even if the API
is unchanged. Select the final deployment checksum/address only afterward.

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
