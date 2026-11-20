# Tooling Changelog

Track compatible CLI, developer-tool, documentation, and off-chain token
registry changes here. These entries must leave validator execution and
contract-observable results and gas unchanged for identical chain inputs.
State-machine changes belong in [CHANGELOG.md](CHANGELOG.md).

Add pending changes under Unreleased in the same PR that implements them,
with a PR link and the user-visible behavior. Call out changes to CLI arguments,
JSON output, or alias meanings explicitly. Consensus compatibility does not
imply that a CLI interface change is backward-compatible for scripts.

During release preparation, move only entries present in the approved release
commit into a version section such as `## v2.23.1`. Keep Unreleased for later
work. Use that version section for the patch release notes. Tag compatible
patches from `release/v2.23`, independently of coordinated chain releases.

## Unreleased

### CLI

- Rewrite the `nibid --help` output. It lists one `Commands`
  section in a fixed order (`tx`, `q`, `keys`, `config`, `status`, `debug`,
  `localnet`, `node`, `help`, `version`), then the flags. The generic usage
  block and the `Aliases: nibid, nibiru` line are gone, and the root command
  has no alias (the `nibiru` command is the npm launcher). Subcommand help
  keeps its usage lines. Group the subcommands of `nibid tx` and `nibid q`
  into everyday modules, offline signing (`tx` only), and the rest. No command
  moves. The root description is "Nibiru CLI: send transactions, query the
  chain, and run a node."
- Make `q` the listed name of the query command, with `query` as its alias.
  Both spellings work everywhere, so `nibid query bank balances` is
  unchanged.
- Hide the flags `--log_format`, `--log_level`, `--log_no_color`, and
  `--trace` from help by default, so they stop repeating under every
  subcommand. Add flag `--help-verbose` on every command to list them. The
  flags parse exactly as before, so service files and cosmovisor setups that
  pass `--log_level` need no change. Flags `-h` and `--home` stay visible.
- Add parent command `nibid node` with `start`, `init`, `export`, `genesis`,
  `prune`, `rollback`, `bft`, `testnet`, and `evm-tx-index`. Every
  previous top-level form, such as `nibid start` and
  `nibid tendermint show-node-id`, keeps its arguments and behavior as a
  hidden alias and prints no deprecation warning. Hidden commands drop out of
  `nibid --help` only.
- Name the consensus command group `nibid node bft` (Nibiru BFT). `tendermint`
  is its alias under `node`. The hidden top-level command keeps the name
  `tendermint`, with `bft` as an alias, so `nibid tendermint show-node-id` and
  `nibid --home H tendermint show-node-id` work unchanged. The `tendermint`
  description is "Nibiru BFT consensus commands".
- Replace the SDK `debug` registration with `nibid debug <input>`, which
  detects the input kind and prints JSON with a `detected` field. Add
  subcommands `debug tx`, `debug msg`, `debug abci-query` (JSON-RPC bodies and
  batches, any gRPC path, nested Simulate and BroadcastTx transactions), and
  `debug stargate`. The SDK subcommands `debug addr`, `debug pubkey`,
  `debug pubkey-raw`, `debug raw-bytes`, and `debug prefixes` keep their
  output.
- Move `build-stargate-wasm` to `debug stargate`. The top-level
  `build-stargate-wasm` stays as a hidden alias with byte-identical output.
  Hidden command `base64-decode` also stays, with identical output for valid
  input. With no argument, it returns a usage error instead of panicking.
  Command `base64-decode` is not removed, so scripts that call it, such as
  `ud cw3`, need no change.
- Install the Nibiru binary as `nibiru`, with `nibid` kept as a compatible
  name. `just install` writes `nibiru` to the Go bin directory and a `nibid`
  symlink beside it. `just build` writes `build/nibiru` and a `build/nibid`
  symlink. The container images install `/usr/local/bin/nibiru` with a
  `nibid` symlink, and their entrypoint is `nibiru`. Release archive names and
  contents are unchanged: each archive holds a regular file named `nibid`.
  Existing `nibid` commands, cosmovisor layouts, and service files keep
  working.
- Print the invoked name in help. `nibiru --help` says `nibiru` and
  `nibid --help` says `nibid`; any other process name prints `nibiru`. The
  default home `~/.nibid`, the keyring service name, and the `version` output
  do not change with the invoked name.
- Add path details to `nibiru version --long`. Both text and JSON output
  (`-o json`) add one field, `binary_paths`, with the same structure, so the
  text output stays one valid YAML document. In JSON it comes after the
  existing fields. Plain `version` output is unchanged. When another file wins
  the name, a one-line note goes to stderr. The report marks the `nibiru` or
  `nibid` file the shell
  runs first when it is not the running binary. A `nibid` symlink to the
  running file is not reported as a conflict.
- Warn at the end of `just install` when `nibiru` or `nibid` on `PATH` is a
  different file than the one installed. The install still succeeds. The
  warning starts with a plain marker line, `nibiru-path-shadowed name=<name>
  runs=<path> installed=<path>`, one per name. When the installed build wins
  and another copy exists, the script prints `nibiru-path-ok name=<name>
  runs=<path> other=<path>` instead.
- Print `nibiru` instead of `nibid` in the command examples of help text when
  the binary runs as `nibiru`.
- Match both `nibid` and `nibiru` in process checks (`pgrep -x`) in the
  `justfile`, `x/nutil/local_blockchain.go`, and the core test workflow.
- Make `nibiru debug` show every reading of an ambiguous input. It prints
  `"detected": "ambiguous"` with a `readings` object: `contract_addr`,
  `ed25519_pubkey`, and `tx_hash` for 32 bytes, and one `hex` and `base64`
  reading for a string valid as both. `--as <kind>` still gives one answer.
- Accept a 33-byte value as a public key in `nibiru debug` only when it is a
  valid secp256k1 point, in `--as pubkey`, automatic detection, and readings.
- Apply `nibiru debug --as <kind>` to both encodings of a string that is valid
  as hex and base64, so every reading of the automatic output is selectable.
  One match is the answer, two give `ambiguous` with `hex` and `base64`
  readings, and none is an error that names the kind.
- Name addresses `addr` in `nibiru debug` output: kind `addr` (in `detected`,
  `candidates`, and `--as addr`; `--as address` stays as an undocumented
  alias), field `eth_addr` instead of `evm`, and field `addr` instead of
  `address` in public key output. These commands are new in this PR, so no
  released output changes.
- Check both `nibid` and `nibiru` node processes, for the target home only,
  in the `localnet` script before it resets chain data. The script waits for
  them to exit and aborts if one stays alive.
- Reword the `testnet` description from "simapp testnet" to "multi-node local
  testnet".
  [PR #2929](https://github.com/SaiDotFun/nibiru/pull/2929).
- Remove `nibiru tx devgas register`, `update`, and `cancel`. The chain
  rejects their messages as part of the `x/devgas` shutdown. The read-only
  `nibiru q devgas` commands stay.
  [PR #2930](https://github.com/SaiDotFun/nibiru/pull/2930).

### Token Registry

- Add Sai USDC at `0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190` and its
  bank denom. Label Stargate `USDC.e` as legacy without changing its identifier
  or symbol. These are off-chain registry updates; node behavior is unchanged.
  [PR #2908](https://github.com/NibiruChain/nibiru/pull/2908),
  [PR #2909](https://github.com/NibiruChain/nibiru/pull/2909).
