# x-consensus

x-consensus is the CosmWasm contract that owns Nibiru BFT's consensus params.
It takes over the role the x/consensus module used to have. It holds the
block, evidence and validator params that Nibiru BFT enforces, plus
`min_block_commit_duration_ms`, a chain-wide commit wait that replaces each
node's `config.toml` `timeout_commit`.

The x/consensus store still exists, but only as the per-block copy that
baseapp and Nibiru BFT read. Its `MsgUpdateParams` is blocked by a circuit
breaker (`app/xconsensus.go`), so x-consensus is the only way to change
consensus params.

## How it works

- **Who can change it.** Only the current x/sudo root. The contract asks
  x/sudo for the root on every execute, through the Stargate query
  `/nibiru.sudo.v1.Query/QuerySudoers`, and stores no owner. A root rotation
  takes effect at once. An empty root or a failed query rejects the message.
- **When a change applies.** The x/xconsensus EndBlocker runs after every
  other module. It queries `params` under a fixed gas limit, validates the
  result with Nibiru BFT's own rules, and copies it into the x/consensus
  store, which baseapp returns to Nibiru BFT from the same EndBlock. A change
  executed in block H applies from block H+1, for the app and the engine
  together. If the query fails or the values are invalid, the store keeps
  its last good params and the chain keeps running.
- **Where it comes from.** The x/xconsensus module creates it with
  `instantiate2` from the embedded build `x/xconsensus/x_consensus.wasm`:
  at genesis for new chains, and in the v2.22 upgrade handler for existing
  chains. The creator is the module address
  `authtypes.NewModuleAddress("xconsensus")` and the salt is `x-consensus`,
  so the address depends only on the code checksum.
- **Migration.** The contract is its own Wasm admin. The root migrates it by
  executing `migrate` on the contract, which then migrates itself, so
  migration rights follow the live root like param updates. The chain also
  adds the contract to the x/sudo `wasm_deployer` role, which the mainnet
  Wasm deployer guard requires of whoever performs a migration. Governance stays
  the break-glass: Nibiru's Wasm keeper lets a gov proposal migrate any
  contract regardless of admin (`x/wasm/keeper/authz_policy.go:66`).
- **Version.** The app version (`version.app`) is not here. Nibiru BFT copies
  it into every block header and x/upgrade owns it, so the Go module copies
  it unchanged. There is no cw2 record either; the code checksum identifies
  the deployed version.

## Params

| Param | Meaning for Nibiru BFT and the chain | Rule |
| --- | --- | --- |
| `block.max_bytes` | Largest block a proposer may build, in bytes, including header, last commit and evidence. Nodes reject larger proposals, and the mempool rejects a transaction too big for any block. Also the cap for `evidence.max_bytes`. | (0, 100 MiB] |
| `block.max_gas` | Total gas the transactions in one block may request. `-1` means no limit. Used by the mempool, proposal building, the ante handler, and baseapp's block gas meter. `0` leaves no room for any transaction that requests gas. | ≥ -1 |
| `evidence.max_age_num_blocks` | Evidence age limit in blocks. With block pruning on, the node also keeps at least this many recent blocks. | > 0 |
| `evidence.max_age_duration_ns` | Evidence age limit in nanoseconds. Evidence is too old only when it exceeds both age limits; within either, it is accepted and slashed. | > 0 |
| `evidence.max_bytes` | Most evidence bytes one block may carry. A cap, not a reservation: transactions use the rest. | [0, `block.max_bytes`] |
| `validator.pub_key_types` | Key types validators may use for consensus. x/staking rejects new validators with other types, and Nibiru BFT halts on a power update for a validator whose type is not listed. Never remove a type a bonded validator uses. | non-empty, `ed25519` or `secp256k1` |
| `min_block_commit_duration_ms` | Minimum time from the commit decision to the start of the next height; see below. `null` means each node keeps its `config.toml` `timeout_commit`. | 0 to 30000, or `null` |

The code comments in [`src/msg.rs`](./src/msg.rs) cite the Nibiru BFT and
Cosmos SDK lines behind each claim.

### `min_block_commit_duration_ms`

The commit decision is the moment a node first has precommits for a block
from more than two thirds of voting power. The node schedules the next height
at that time plus this value. Executing and saving the block happen inside
the window, so it is not a pause added afterwards.

```text
CommitTime = this node first has precommits from >2/3 of voting power
next height scheduled at CommitTime + min_block_commit_duration_ms
  |-- Commit: execute and save the block --|-- NewHeight: wait --|
if execution takes longer, the next height starts as soon as it finishes
```

When set, it replaces `timeout_commit` on every node. It is a floor, not an
exact length: node-local `skip_timeout_commit` (once every validator's
precommit has arrived) and the catch-up rule (precommits from more than two
thirds of voting power for the next height) can still start the next height
earlier. The 30 s cap exists because a longer wait would make the chain look
halted.

## Messages

Instantiate ([fixture](./testdata/instantiate_msg.json)). Sent only by the
x/xconsensus module; it needs no root.

```json
{
  "params": {
    "block": { "max_bytes": "22020096", "max_gas": "-1" },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration_ns": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": { "pub_key_types": ["ed25519"] },
    "min_block_commit_duration_ms": null
  }
}
```

Execute (x/sudo root only):

- `{"update_params": {"params": <params>}}` replaces all params at once
  ([fixture](./testdata/update_params_msg.json)). There is no partial
  update, because some rules span fields; send the current params with your
  change.
- `{"migrate": {"code_id": 42, "msg": "e30="}}` migrates the contract to
  code 42. `msg` is the base64 JSON for the new code's `migrate` entry point
  (`e30=` is `{}`). The Wasm keeper checks the migration against the
  contract, which is the caller:
  - The new code's instantiate permission must allow the x-consensus
    address. Store it with, for example,
    `nibid tx wasm store new.wasm --instantiate-anyof-addresses <x-consensus address>`
    (or `--instantiate-everybody`). This is an operator step.
  - The mainnet Wasm deployer guard requires the caller to hold the x/sudo
    `wasm_deployer` role. The chain grants it to x-consensus when it creates
    the contract, on every network, so no manual step is needed. The role
    gives the contract nothing beyond self-migration, since it can only
    migrate or re-admin itself, and only on the live root's order.
- `{"update_admin": {"admin": "nibi1..."}}` hands the Wasm admin to another
  address. After that, `migrate` through the contract fails and only that
  address or governance can migrate. Param updates are unaffected.

Query:

- `{"params": {}}` returns the params object above.

Migrate entry point: `{}`. It keeps the stored params.

## JSON conventions

- Go int64 fields are JSON strings (CosmWasm `Int64`), so values above 2^53
  stay exact in every decoder.
- `max_age_duration_ns` is in nanoseconds, the unit of Go `time.Duration`,
  so it converts without loss.
- `min_block_commit_duration_ms` is a JSON number from 0 to 30000, or `null`
  (or absent) for unset.
- Unknown fields are rejected, so a typo fails instead of being dropped.

## Validation

`src/validate.rs` ports `ConsensusParams.ValidateBasic` from
`bft/types/params.go` rule for rule, with line citations, and adds the 0 to
30000 ms bound on `min_block_commit_duration_ms`. Nibiru BFT halts on params
that fail `ValidateBasic`, so the contract rejects them first, and the Go
module checks again before copying.

## Commands

```bash
cargo test -p x-consensus
just rs wasm-all release   # optimized build into wasm-contracts/artifacts
just rs wasm-check
```
