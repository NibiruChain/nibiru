---
order: 1
footer:
  newsletter: false
description: >
  Use the Nibiru CLI to query chain data, send transactions, and run nodes.
  This guide introduces its configuration, commands, and flags.
---

# Nibiru CLI (Command-Line Interface)

{{ $frontmatter.description }}

## Nibiru CLI Guides

1. [Nibiru CLI - How to Install the Nibiru CLI](./nibid-binary.md)
2. [Nibiru CLI - Creating Fungible Tokens](./tf.md)

The native binary is named `nibiru`, and `nibid` stays as a compatible name
(a symlink in source builds and container images). Both names run the same
program, and help prints the name you invoked. Bun and npm installs expose
`nibiru`, which forwards all arguments to the native binary. The examples in
these guides use `nibid`; replace it with `nibiru` as you prefer.

## Command List

`nibid --help` prints one command list in a fixed order:
`tx`, `q` (alias `query`), `keys`, `config`, `status`, `debug`, `localnet`,
`node`, `help`, and `version`. Commands that run a node live under
`nibid node`, for example `nibid node start` and
`nibid node genesis validate-genesis`. The previous top-level forms, such as
`nibid start`, keep working as hidden aliases.

Help hides the log and trace flags (`--log_format`, `--log_level`,
`--log_no_color`, `--trace`) by default. They still work when passed. Add
`--help-verbose` to any command to list them.

`nibid tx --help` and `nibid q --help` list the everyday modules first
(`bank`, `wasm`, `evm`, `staking`, `gov`, `distribution`, `tokenfactory`,
`ibc-transfer`). `nibid tx --help` then lists the offline signing commands.

## Decode Encoded Data

`nibid debug <input>` detects what the input is and prints JSON with a
`detected` field. It reads stdin when the argument is `-`, or when no argument
is given and stdin is piped. It works offline and accepts:

| Input                                                 | Detected as        |
| ----------------------------------------------------- | ------------------ |
| JSON-RPC body or batch (`abci_query`, `broadcast_tx_*`) | `abci-query`     |
| JSON with `type_url` and `value`, or a `stargate` wrapper | `any`          |
| Base64 or hex transaction                             | `tx`               |
| Bech32 (`nibi1...`) or `0x` address                   | `address`          |
| Bech32 pubkey, proto-JSON pubkey, raw 33-byte key     | `pubkey`           |
| `[10 21 13 255]`                                      | `raw-bytes`        |
| Other valid protobuf                                  | `unknown-protobuf` |

A 32-byte value fits several kinds (transaction hash, Wasm contract address,
raw key). The output lists the candidates. Pass `--as <kind>` to choose one.

Explicit subcommands decode one kind: `debug tx`, `debug msg`,
`debug abci-query`, `debug stargate`, `debug addr`, `debug pubkey`,
`debug pubkey-raw`, `debug raw-bytes`, and `debug prefixes`. Command
`debug stargate` replaces `build-stargate-wasm`, which stays as a hidden alias
with identical output.

## Working Directory

The CLI stores configuration, blockchain data, and local keyring files under
`$HOME/.nibid` by default. Use the `--home` flag to select another directory.

## Connect to a Full Node

By default, `nibid` connects to the RPC endpoint at `tcp://localhost:26657`.
Use the `--node` flag to connect to a full node on another machine.

## Global Flags

### Query Commands

Query commands accept these flags. Run `nibid query <module> <command> --help`
for the flags of a specific command.

| Name, shorthand | Type   | Default value         | Description                                              |
| --------------- | ------ | --------------------- | -------------------------------------------------------- |
| --chain-id      | string |                       | Network chain ID                                         |
| --node          | string | tcp://localhost:26657 | RPC endpoint for the full node                           |
| --grpc-addr     | string |                       | gRPC endpoint to use for this chain                      |
| --grpc-insecure | bool   | false                 | Allow gRPC over insecure channels                        |
| --height        | int    |                       | Query state at a specific height (the node may have pruned it) |
| --output, -o    | string | text                  | Output format: text or json                              |

Global flags `--home`, `--log_format`, `--log_level`, `--log_no_color`, and
`--trace` apply to every command. `--trace` is a boolean.

### Transaction Commands

Transaction commands accept these flags. Run `nibid tx <module> <command> -v --help`
to print all of them.

| Name, shorthand       | Type   | Default               | Description |
| --------------------- | ------ | --------------------- | ----------- |
| --account-number, -a  | uint   | 0                     | Account number of the signing account (offline mode only) |
| --broadcast-mode, -b  | string | sync                  | Broadcast mode: sync or async |
| --chain-id            | string |                       | Network chain ID |
| --dry-run             | bool   | false                 | Simulate the transaction without broadcasting it |
| --fee-granter         | string |                       | Account that grants the fees for the transaction |
| --fee-payer           | string |                       | Account that pays the fees instead of the signer |
| --fees                | string |                       | Fees to pay along with the transaction, for example `10unibi` |
| --from                | string |                       | Name or address of the private key that signs |
| --gas                 | string | 200000                | Gas limit per transaction; use `auto` to estimate it |
| --gas-adjustment      | float  | 1.05                  | Multiplier applied to the simulated gas estimate |
| --gas-prices          | string | 0.025unibi when `--fees` is omitted | Gas prices in decimal format, for example `0.1unibi` |
| --generate-only       | bool   | false                 | Write an unsigned transaction to standard output |
| --help, -h            | bool   | false                 | Print help message |
| --keyring-backend     | string | os                    | Keyring backend: os, file, kwallet, pass, test, or memory |
| --keyring-dir         | string | `--home` directory    | Directory of the client keyring |
| --ledger              | bool   | false                 | Use a connected Ledger device |
| --node                | string | tcp://localhost:26657 | RPC endpoint for the full node |
| --note                | string |                       | Note to add to the transaction (previously `--memo`) |
| --offline             | bool   | false                 | Run without connecting to a node |
| --output, -o          | string | json                  | Output format: text or json |
| --sequence, -s        | uint   | 0                     | Sequence number of the signing account (offline mode only) |
| --sign-mode           | string |                       | Signing mode: direct, amino-json, or direct-aux |
| --timeout-height      | uint   | 0                     | Block height after which the transaction can't be committed |
| --yes, -y             | bool   | false                 | Skip the transaction confirmation prompt |

### Module Commands

| Subcommand                            | Description                                                    |
| ----------------------------------------- | ------------------------------------------------------------------ |
| [devgas](../../concepts/arch/advanced/devgas.md#command-line-interface) | Devgas subcommands for smart contract usage.                       |
| [bank](../../concepts/arch/advanced/cosmos-sdk/bank.md#cli)     | Bank subcommands for managing assets.                              |
| [keys](../../concepts/arch/advanced/keys.md#available-commands)     | Keys subcommands for managing local tendermint keystore.           |

<!-- | [evm](../arch/advanced/evm.md#cli)     | Ethereum Virtual Machine (EVM) Nibiru CLI commands  | -->
<!-- | [wasm](../arch/advanced/wasm.md#cli)     | Wasm subcommands for enabling CosmWasm smart contracts execution.  | -->
