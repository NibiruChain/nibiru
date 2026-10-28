# x-consensus

Holds every Nibiru BFT consensus parameter plus the chain-wide
`min_block_commit_duration_ms`. It replaces the storage role of the
x/consensus module. A Go adapter in `nibid` queries `params` once per block
at EndBlock, so a change applies from the next block for both the app and
the engine. The owner is the x/sudo root. `nibid` embeds the optimized build
from `x/xconsensus/x_consensus.wasm`.

## Messages

Instantiate ([fixture](./testdata/instantiate_msg.json)):

```json
{
  "owner": "nibi1qqx5reauy4glpskmppy88pz25qp2py5yxvpxdt",
  "params": {
    "block": { "max_bytes": "22020096", "max_gas": "-1" },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration_ns": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": { "pub_key_types": ["ed25519"] },
    "version": { "app": "0" },
    "min_block_commit_duration_ms": null
  }
}
```

Execute (owner only):

- `{"update_params": {"params": <params>}}` replaces all params
  ([fixture](./testdata/update_params_msg.json)).
- `{"update_owner": {"owner": "nibi1..."}}` transfers ownership.

Query:

- `{"params": {}}` returns the params object above.
- `{"owner": {}}` returns `{"owner": "nibi1..."}`.

## JSON conventions

- Go int64 and uint64 fields are JSON strings (CosmWasm `Int64` and
  `Uint64`), so values above 2^53 stay exact in every decoder.
- `max_age_duration_ns` is in nanoseconds, the unit of Go `time.Duration`,
  so it converts without loss.
- `min_block_commit_duration_ms` is a JSON number from 0 to 30000, or `null`
  (or absent) for unset. Unset means each node keeps its `config.toml`
  `timeout_commit`.

## Validation

`src/validate.rs` ports `ConsensusParams.ValidateBasic` from
`bft/types/params.go` rule for rule, with line citations, and adds the 0 to
30000 ms bound on `min_block_commit_duration_ms`.

## Commands

```bash
cargo test -p x-consensus
just rs wasm-all release   # optimized build into wasm-contracts/artifacts
just rs wasm-check
```
