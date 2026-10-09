# Eris v2.21 recovery fixture

These fixtures capture the deployed Eris hub and its public state on Nibiru
mainnet. They support an offline replay of the v2.21 upgrade handler, including
the sudo migration and deployer-role seeding.

| Capture field | Value |
| --- | --- |
| Chain | `cataclysm-1` |
| Height | `47554727` |
| Block time | `2026-10-09T04:59:56.720420894Z` |
| Block hash | `487D702C0BD6613475C3915C64FD337D05D6B82610875BCD31365A1A63A46DFF` |
| Endpoint | `https://rpc.nibiru.fi:443` |
| Eris code ID | `132` |
| Uncompressed Wasm SHA-256 | `e1c2be31ae008a015efa16b51f74ca1a014f4b1d0952da12ba3f60aaffa66321` |
| Storage entries | `1752` |
| Uncompressed bytecode size | `768931` bytes |

File `hub.wasm.gz` contains the bytecode retrieved with `nibid q wasm code`,
compressed with a zero gzip timestamp. File `snapshot.json` contains contract
and code metadata, code history, every raw storage page, the attacker request,
and the hub, attacker, and Treasury CW3 native balances. Every query uses the
same height. Raw storage keys use hexadecimal encoding; values use base64.

The test imports the real Eris code and storage at its mainnet address. It uses
the existing CW3 test bytecode for the recipient's metadata, with no CW3 state,
because the withdrawal delivers a Bank send and does not execute the recipient.
Only relevant native balances and the contract state are replayed; this fixture
is not an export of the entire chain or its validator state.

## Capture and test

From the repository root, refresh the fixture with public read-only queries:

```bash
python3 app/upgrades/testdata/eris-v2.21/capture.py
```

To reproduce this height, pass `--height 47554727` and an archive `--node` if the
standard RPC has pruned it. The script rejects a different chain or Eris code
checksum and never signs or broadcasts a transaction. Review refreshed state
before changing test expectations.

Run the focused replay and quarantine checks:

```bash
go test ./app/upgrades ./app/ante -run 'TestUpgrade221Eris|TestUpgrade2_21|TestAnteDecIncidentQuarantine' -count=1 -v
```

After `just build` downloads and verifies the pinned static library, run the
full upgrade and ante packages against the release runtime using the Alpine
musl toolchain. This command is for Linux ARM64; on AMD64, change the library
directory to `linux_amd64`:

```bash
docker run --rm \
  -v "$PWD:/nibiru" \
  -v "$(go env GOPATH)/pkg/mod:/go/pkg/mod" \
  -v "$(go env GOCACHE):/root/.cache/go-build" \
  -w /nibiru -e CGO_ENABLED=1 \
  -e 'CGO_LDFLAGS=-L/nibiru/temp/wasmvm/v1.13.1/lib/linux_arm64 -lm' \
  golang:1.27.0-alpine3.24 sh -c \
  "apk add --no-cache build-base linux-headers && \
   go test -buildmode=pie -tags 'netgo osusergo ledger static pebbledb muslc' \
   -ldflags \"-linkmode=external -extldflags '-Wl,-z,muldefs -static-pie -z noexecstack'\" \
   ./app/upgrades ./app/ante -count=1 -v"
```

## Recovery expectations

The attacker owns `32298107051806` of batch 172's `32298147847709` shares. Its
payout is integer division of `51100727292767 * 32298107051806 / 32298147847709`,
which yields `51100662747260 unibi`, or `51,100,662.747260 NIBI`.

The other request retains `40795903` shares and `64545507 unibi`. If that user
withdraws first, the attacker's payout becomes `51100662747261 unibi` because
Eris truncates each withdrawal separately. The contract computes the payout
from activation-time state rather than the handler hardcoding either amount.

The handler queries only the native configuration and reviewed request before
execution to decide whether this withdrawal is appropriate. After those checks,
it executes the contract and relies on Eris and the Bank keeper for settlement.
Balance deltas, supply, request removal, and remaining batch arithmetic are
assertions in the replay tests. The handler performs no bank balance reads or
post-execution queries. Its recovery event copies the payout from Eris's event
when available; missing payout metadata does not reject a successful execute.

The replay checks the destination balance, native supply, unchanged attacker
balances, and all unrelated contract storage. It also exercises repeated
execution, a prior legitimate withdrawal, non-mainnet exclusion, unexpected
code and additional claims, immaturity, unreconciled state, insufficient funds,
failed Bank dispatch, and gas exhaustion. Failed recovery attempts discard
contract writes and events while preserving successful deployment grants.

Custom steps are private methods on `Handler_v2_21`. Deployment seeding and
Eris recovery use separate cached stores and gas meters, and both returned
errors and recoverable Go panics reach the outer handler's `upgrade_failure`
event. A failed seed still allows recovery to run. Failure reports use the
parent context so discarding a step preserves its diagnostic events. Required
module migration errors still propagate because the binary needs the migrated
schema.

The cached-step regression test writes state and emits an event before injecting
an error, panic, or gas exhaustion. It verifies rollback, unchanged parent gas,
and successful execution of a later step. The Eris replay also verifies recovery
after a failed deployer seed and the outer failure event for failed recovery.

## Validation recorded on 2026-10-09

- `go test ./app/upgrades ./app/ante -count=1 -v` passed both full packages.
- The Alpine musl command above passed both full packages with the pinned
  WasmVM v1.13.1 static library on Linux ARM64.
- The simplified recovery consumed `263263` SDK gas under the release runtime.
- `just build` completed and `just go-lint` reported zero issues.

An earlier static test run linked with the workstation's glibc toolchain
crashed before tests started. The release-runtime validation uses Alpine musl,
matching `contrib/docker/Dockerfile.release-musl`.

These checks cover the captured Eris state and the v2.21 handler. Final release
candidate rehearsal, cross-architecture comparison, and activation observation
remain in the security epic.
