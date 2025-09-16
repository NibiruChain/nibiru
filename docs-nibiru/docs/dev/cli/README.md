---
order: 1
footer:
  newsletter: false
---

# Nibiru Command-Line Interface (CLI)

An introduction to the using the Nibiru Command-Line Interface (CLI) along with a
brief description of its commands and flags. After installation, you can use the
`nibid` command to send transactions and query the Nibiru blockchain. {synopsis}

## Nibiru CLI Guides

1. [Nibiru CLI - How to Install the Nibiru CLI](./nibid-binary.md)
2. [Nibiru CLI - Creating Fungible Tokens](./tf.md)

## Working Directory

The default working directory for the `nibid` is `$HOME/.nibid`, which is mainly used to store configuration files and blockchain data. The Nibiru `key` data is saved in the working directory of `nibid`. You can also specify the `nibid` working directory by using the `--home` flag when executing `nibid`.

## Connecting to a Full-Node

By default, `nibid` uses `tcp://localhost:26657` as the RPC address to connect to the Nibiru network. This default configuration assumes that the machine executing `nibid` is running as a full-node.

The RPC address can be specified to connect to any full-node with an exposed RPC port by adding the `--node` flag when executing `nibid`

## Global Flags

#### GET Commands

All GET commands have the following global flags:

| Name, shorthand | type   | Default Value | Description                          |
| --------------- | ------ | ------------- | ------------------------------------ |
| --chain-id      | string |               | The network Chain ID                 |
| --home          | string | $HOME/.nibid  | Directory for config and data        |
| --trace         | string |               | Print out full stack trace on errors |
| --log\_format   | string | plain         | Logging format (json \| plain)       |

#### POST Commands

All POST commands have the following global flags:

| Name, shorthand   | type   | Default               | Description
| ----------------- | ------ | --------------------- | -------------------------------------------------------------------------------------------------------------- |
| --account-number  | int    | 0                     | `AccountNumber` to sign the tx
| --broadcast-mode  | string | sync                  | Transaction broadcasting mode (sync \| async \| block)
| --dry-run         | bool   | false                 | Ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it                        |
| --fees            | string |                       | Fees to pay along with transaction
| --from            | string |                       | Name of private key with which to sign
| --gas             | string | 200000                | Gas limit to set per-transaction; set to "simulate" to calculate required gas automatically                    |
| --gas-adjustment  | float  | 1                     | Adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set |
| --gas-prices      | string |                       | Gas prices in decimal format to determine the transaction fee                                                  |
| --generate-only   | bool   | false                 | Build an unsigned transaction and write it to STDOUT 
| --help, -h        | string |                       | Print help message
| --keyring-backend | string | os                    | Select keyring's backend
| --ledger          | bool   | false                 | Use a connected Ledger device
| --memo            | string |                       | Memo to send along with transaction
| --node            | string | tcp://localhost:26657 | specifies `<host>:<port>` for the Tendermint RPC interface endpoint for this chain                                                     |
| --offline         | string |                       | Offline mode (does not allow any online functionality)
| --sequence        | int    | 0                     | Sequence number to sign the tx
| --sign-mode       | string |                       | Choose sign mode (direct \| amino-json), this is an advanced feature                                           |
| --trust-node      | bool   | true                  | Don't verify proofs for responses
| --yes             | bool   | true                  | Skip tx broadcasting prompt confirmation
| --chain-id        | string |                       | The network Chain ID
| --home            | string | $HOME/.nibid          | Directory for config and data
| --trace           | string |                       | Print out full stack trace on errors

### Module Commands

| **Subcommand**                            | **Description**                                                    |
| ----------------------------------------- | ------------------------------------------------------------------ |
| [devgas](../../arch/advanced/devgas.md#cli) | Devgas subcommands for smart contract usage.                       |
| [bank](../arch/advanced/bank.md#cli)     | Bank subcommands for managing assets.                              |
| [keys](../arch/advanced/keys.md#cli)     | Keys subcommands for managing local tendermint keystore.           |

<!-- | [evm](../arch/advanced/evm.md#cli)     | Ethereum Virtual Machine (EVM) Nibiru CLI commands  | -->
<!-- | [wasm](../arch/advanced/wasm.md#cli)     | Wasm subcommands for enabling CosmWasm smart contracts execution.  | -->
