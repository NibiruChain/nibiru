# v2.23 Holder-Set Fixture

`usdc_holder_set.wasm.gz` is test data, not a runtime embed. The upgrade requires
the registry to have been deployed beforehand. Tests decompress these bytes,
upload and instantiate them through the real Wasm keeper, update the set, and
exercise the Go reader. No source-level contract mock substitutes for execution.

Decompressed SHA-256:

```text
ae3840f748807eb93400ea7d0208ae084c002148ed60177fbf276a1ee407003f
```

Temporary source: `wasm-contracts/usdc-holder-set`. Build with Rust 1.86.0 as
specified in that package's README, then compress with timestamp zero:

```sh
python3 - <<'PY'
import gzip
from pathlib import Path
source = Path('target/wasm32-unknown-unknown/release/usdc_holder_set.optimized.wasm')
target = Path('app/upgrades/testdata/v2.23/usdc_holder_set.wasm.gz')
target.write_bytes(gzip.compress(source.read_bytes(), compresslevel=9, mtime=0))
PY
```

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
