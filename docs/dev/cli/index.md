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

The native CLI command is `nibid`. Bun and npm installs expose it as `nibiru`,
which forwards all arguments to the native binary. For those installs, replace
`nibid` with `nibiru` in the examples below and in the linked guides.

## Working directory

The CLI stores configuration, blockchain data, and local keyring files under
`$HOME/.nibid` by default. Use the `--home` flag to select another directory.

## Connect to a full node

By default, `nibid` connects to the RPC endpoint at `tcp://localhost:26657`.
Use the `--node` flag to connect to a full node on another machine.

## Global flags

### Query commands

Query commands use the following global flags:

| Name, shorthand | Type   | Default value | Description                          |
| --------------- | ------ | ------------- | ------------------------------------ |
| --chain-id      | string |               | Network chain ID                     |
| --home          | string | $HOME/.nibid  | Directory for config and data        |
| --trace         | string |               | Print out full stack trace on errors |
| --log\_format   | string | plain         | Log format: json or plain            |

### Transaction commands

Transaction commands use the following global flags:

| Name, shorthand   | Type   | Default               | Description
| ----------------- | ------ | --------------------- | -------------------------------------------------------------------------------------------------------------- |
| --account-number  | int    | 0                     | `AccountNumber` to sign the tx
| --broadcast-mode  | string | sync                  | Broadcast mode: sync, async, or block |
| --dry-run         | bool   | false                 | Simulate the transaction without broadcasting it |
| --fees            | string |                       | Fees to pay along with transaction
| --from            | string |                       | Name of private key with which to sign
| --gas             | string | 200000                | Gas limit per transaction; use "simulate" to estimate it |
| --gas-adjustment  | float  | 1                     | Multiplier applied to the simulated gas estimate |
| --gas-prices      | string |                       | Gas prices in decimal format to determine the transaction fee                                                  |
| --generate-only   | bool   | false                 | Write an unsigned transaction to standard output |
| --help, -h        | string |                       | Print help message
| --keyring-backend | string | os                    | Select keyring's backend
| --ledger          | bool   | false                 | Use a connected Ledger device
| --memo            | string |                       | Memo to send along with transaction
| --node            | string | tcp://localhost:26657 | RPC endpoint for the full node |
| --offline         | string |                       | Run without connecting to a node |
| --sequence        | int    | 0                     | Sequence number to sign the tx
| --sign-mode       | string |                       | Signing mode: direct or amino-json |
| --trust-node      | bool   | true                  | Don't verify proofs for responses
| --yes             | bool   | true                  | Skip the transaction confirmation prompt |
| --chain-id        | string |                       | Network chain ID |
| --home            | string | $HOME/.nibid          | Directory for config and data
| --trace           | string |                       | Print out full stack trace on errors

### Module commands

| Subcommand                            | Description                                                    |
| ----------------------------------------- | ------------------------------------------------------------------ |
| [devgas](../../concepts/arch/advanced/devgas.md#cli) | Devgas subcommands for smart contract usage.                       |
| [bank](../../concepts/arch/advanced/cosmos-sdk/bank.md#cli)     | Bank subcommands for managing assets.                              |
| [keys](../../concepts/arch/advanced/keys.md#cli)     | Keys subcommands for managing local tendermint keystore.           |

<!-- | [evm](../arch/advanced/evm.md#cli)     | Ethereum Virtual Machine (EVM) Nibiru CLI commands  | -->
<!-- | [wasm](../arch/advanced/wasm.md#cli)     | Wasm subcommands for enabling CosmWasm smart contracts execution.  | -->
