# Sai Pulse block-hook integration

Status: Real-artifact application integration passed; network rollout pending.
Code scope: Sai Perps `contracts/sai-pulse`, Nibiru `x/wasm/keeper` integration
tests, and the existing root-controlled `x/sudo` registry configuration.

Sai Pulse is a Sai Perps contract that implements both the Nibiru block-hook
registry and a vault snapshot target. It replaces no Keeper jobs at activation.
The initial production configuration observes the two USDC SLP vaults; stNIBI
vaults are deprecated and excluded.

```text
EndBlock at height divisible by interval
  -> x/wasm queries Pulse end_block_plan
  -> Pulse returns a JSON array of self-directed sudo calls
  -> one snapshot call per configured SLP
  -> three smart queries against that SLP
  -> wasm-sai/pulse/vault_snapshot in EndBlock events
```

No transaction is required at the snapshot height. The configured interval is
measured in blocks, defaults to 1,000, and uses globally aligned heights. Pulse
starts paused. Its BeginBlock plan is always empty. The chain's Wasm EndBlock
handler runs after the native oracle and EVM handlers.

## Contract interface

Instantiation uses the main CW3 as owner, with the same address supplied as the
Wasm contract admin. The upload signer does not become the configuration owner.

```json
{
  "owner": "nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v",
  "slp_addrs": [
    "nibi193m2a00pmdsvkcvugrfewqzhtq6k0srkjzvxp2sk357vlpspx5vqxu8d7p",
    "nibi1waf5c8z55qvjay4de8wkm9cxyt6wa8zdnrvlexjrq77lqgqf258q3yn7l8"
  ],
  "interval": 1000
}
```

Queries are `{"config":{}}`, `{"ownership":{}}`,
`{"begin_block_plan":{}}`, and `{"end_block_plan":{}}`.
The registry response must be a bare array, not an object wrapper. On a due,
enabled height it has this shape:

```json
[
  {
    "contract_addr": "<Pulse address>",
    "msg": {"snapshot": {"slp_addr": "<configured SLP address>"}}
  }
]
```

Owner-only execute messages are `{"pause":{}}`, `{"resume":{}}`,
`{"update_config":{"slp_addrs":["<SLP>"],"interval":1000}}`,
and the standard `nibiru_ownable` ownership actions. Omitted update fields retain
their values. Interval zero, duplicate or invalid addresses, and more than 32
vaults are rejected. Pulse can dispatch only its own snapshot operation.

The sudo handler checks pause state, schedule, and configured vault membership.
Each call queries `vault_snapshot`, `available_assets`, and `get_revenue_info`.
An ordinary query failure returns an error before any snapshot event is emitted.
The host records that dispatch failure and continues with later vault calls.
There is no retry before the next interval and no partial snapshot.

## Event contract for indexers

The custom event type is `sai/pulse/vault_snapshot`; the host exposes it as
`wasm-sai/pulse/vault_snapshot`. Its application attributes are:

| Attribute | Meaning |
| --- | --- |
| `slp_addr` | The observed SLP contract address. |
| `data` | One JSON object containing the three complete query responses. |

The host adds `_contract_address`, identifying Pulse rather than the SLP.

```json
{
  "vault_snapshot": {
    "tvl": "0",
    "market_cap": "0",
    "share_price": "1",
    "collateralization_p": "1",
    "total_supply": "0",
    "total_liability": "0",
    "total_rewards": "0",
    "epoch": 1,
    "epoch_start": "1792454576666980654"
  },
  "available_assets": "0",
  "revenue_info": {
    "net_profit": "0",
    "rewards": "0",
    "closed_pnl": "0",
    "liabilities": "0",
    "current_epoch_positive_open_pnl": "0"
  }
}
```

This is an illustrative empty-vault payload. `data` contains nested JSON values,
not three JSON-encoded strings. Monetary integers use collateral base units,
share supply uses share base units, decimal values retain their native string
encoding, and `epoch_start` is nanoseconds. Preserve signed values and avoid
floating-point conversion. There is no schema-version attribute. An incompatible
event format must use a new event type.

The containing block supplies height and time. A later Keeper integration must
use chain, height, lifecycle phase, and event position for replay identity and
accept block events without a transaction hash or message row. Ordinary tx-only
history joins will not retain these events. Its projection maps snapshot `epoch`
to `current_epoch`; all other existing vault-history fields come from the named
response objects. Old history before activation still needs the existing
historical query path.

## Verify a built artifact

Use CI artifacts from the exact Sai Perps source commit under review. Preserve
the run URL, source commit, and artifact checksums. The directory must contain
`sai_pulse.wasm`, `vault.wasm`, and `vault_token_minter.wasm` from the same build.
Generated Wasm is local build output and is not committed here.

```bash
just test-sai-pulse /absolute/path/to/verified/artifacts
```

This opt-in Go integration test loads real Wasm into an in-memory Nibiru app and
checks pause, aligned cadence, BeginBlock exclusion, same-state query parity,
ordinary failure isolation, and registry clearing. It logs code hashes, consumed
SDK gas, event payload bytes, and elapsed execution time. Setup uses two real
vaults with separate token minters, deposits, and trading proceeds, including
negative liabilities. This synthetic state is not a mainnet performance
benchmark. Pulse's contract tests cover numeric serialization separately. The
test logs `snapshot_fixture` JSON records for the later indexer integration.

The initial verified build came from [Sai Perps PR 454](https://github.com/NibiruChain/sai-perps/pull/454),
head `49adc56e51ad7bbbb1e2a5d0029e39a107472ab5`,
[artifact run 39880735839](https://github.com/NibiruChain/sai-perps/actions/runs/39880735839),
tested merge commit `e95d8b9865693edcc8554fa6f7581caa5512e06b`.
Pulse SHA-256 was
`45f6b694ad69ce2661437d96125fcec0beac154f7024b401a6f8a18b0bffbd7f`.
The real-artifact test passed with 1,156,911 SDK gas for the scheduled two-vault
EndBlock, approximately 3–4 ms in the local test, and 381 bytes per JSON payload.
These are synthetic-fixture measurements, not production acceptance results.

## Activation and rollback

Before any registration, confirm the target network, node version/source commit,
the root CW3, current registry, contract owner/admin, and artifact code hash.
The public mainnet node reported application version `2.21.0`, commit
`18d74f6ea2e8f43a7f0440a491c05a79e0dde4b2`, during initial investigation.
That source includes this registry protocol. This does not replace a fresh
deployment-time check.

The registry address is not returned by `nibiru q sudo state`. Its collections
storage key is namespace byte `03` followed by the eight-byte zero item key:

```bash
curl -fsS 'https://rpc.nibiru.fi/abci_query?path=%22/store/sudo/key%22&data=0x030000000000000000'
```

A null value means unset. A nonempty value needs decoding with the matching
chain codec and reconciliation before replacing it. Do not infer the registry
state from the sudoers query alone.

After localnet and testnet verification, upload missing bytecode only after
checking all existing code pages for its hash. Instantiate paused with the CW3
owner/admin and the reviewed USDC configuration. Record code ID, address, hashes,
source revision, and successful transaction receipts.

Prepare a CW3 proposal containing these ordered actions:

1. Register Pulse through `/nibiru.sudo.v1.MsgEditSudoers`, with CW3 as sender,
   action `edit_wasm_block_hooks_contract`, and `contracts: ["<Pulse address>"]`.
2. Execute `{"resume":{}}` against Pulse with no funds.

Build the first action with `nibiru build-stargate-wasm` and the chain's protobuf
codec. Simulate both actions together through the existing CW3 proposal tools
before submission. Registration is root-only, so a direct upload signer cannot
perform it. Preserve the reviewed proposal and subsequent vote/execute receipts.

Verify both vault events at two successive scheduled heights using the RPC
`block_results` EndBlock events, not transaction search. Record event counts,
payloads, query parity at the sampled height, indexer visibility, and block cost.
Do not disable the Keeper history sampler before its separate consumer change
has been implemented and reconciled.

The host currently isolates failed writes/events but does not impose a separate
per-dispatch gas cap. The 32-call and payload limits are not a gas budget. Test
the deployed vault code and failure cases before activation. If execution cannot
be bounded adequately, leave Pulse paused and resolve the host requirement in
a separately reviewed chain upgrade.

For rollback, the owner can execute `{"pause":{}}`, or root can clear the registry
using the same registration message with `contracts: [""]`. Clearing registration
stops all registry dispatches; pausing Pulse makes its plans empty. Both changes
require CW3 execution on mainnet.
