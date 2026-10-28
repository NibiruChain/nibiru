# x-consensus

Holds the Nibiru BFT block, evidence and validator consensus params plus
the chain-wide `min_block_commit_duration_ms`. It replaces the storage role
of the x/consensus module. A Go adapter in `nibid` queries `params` once per
block at EndBlock, so a change applies from the next block for both the app
and the engine. `nibid` embeds the optimized build from
`x/xconsensus/x_consensus.wasm`.

Only the current x/sudo root may update, checked on every execute with the
Stargate query `/nibiru.sudo.v1.Query/QuerySudoers`, so a root rotation
applies at once and no owner is stored. An empty root or a failed query
rejects the update. The app version (`version.app`) is not here: it belongs
to x/upgrade, and the Go adapter copies it unchanged.

## Messages

Instantiate ([fixture](./testdata/instantiate_msg.json)):

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

- `{"update_params": {"params": <params>}}` replaces all params
  ([fixture](./testdata/update_params_msg.json)).

Query:

- `{"params": {}}` returns the params object above.

## JSON conventions

- Go int64 fields are JSON strings (CosmWasm `Int64`), so values above 2^53 stay exact in every decoder.
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
