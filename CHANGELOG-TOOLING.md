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

- Group `nibid --help` into Everyday Commands (`tx`, `query`, `keys`,
  `config`, `status`, `localnet`, `debug`), Run a Node (`node`), and Other
  Commands. Group the subcommands of `nibid tx` and `nibid query` into
  everyday modules, offline signing (`tx` only), and the rest. No command
  moves. The root description now reads "Nibiru CLI: send transactions, query
  the chain, and run a node."
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
- Reword the `testnet` description from "simapp testnet" to "multi-node local
  testnet".
  PR link: TBD (placeholder, replace before merge).

### Token Registry

- Add Sai USDC at `0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190` and its
  bank denom. Label Stargate `USDC.e` as legacy without changing its identifier
  or symbol. These are off-chain registry updates; node behavior is unchanged.
  [PR #2908](https://github.com/NibiruChain/nibiru/pull/2908),
  [PR #2909](https://github.com/NibiruChain/nibiru/pull/2909).
