---
order: 1
metaTitle: "Guide: Building Apps with NibiJS"
---

# NibiJS

A brief guide on Nibiru's APIs, docs, and resources to broadcast transactions and query the chain.  {synopsis}

#### Table of Contents

- [Nibiru JS (`nibijs`): Installation](#nibiru-js-nibijs-installation)
- [Nibiru JS (`nibijs`): Query and Tx Clients](#nibiru-js-nibijs-query-and-tx-clients)
- [Nibiru JS (`nibijs`): Common Queries](#nibiru-js-nibijs-common-queries)
    - [Transaction Hashes](#transaction-hashes)
    - [BlockResponse.block.header from `querier.tm.block(height)`](#blockresponseblockheader-from-queriertmblockheight)
- [Wallets](#wallets)
- [Test Network Tokens (Faucet)](#test-network-tokens-faucet)
- [Running a Full Node](#running-a-full-node)

## Nibiru JS (`nibijs`): Installation

```bash
# 1 - Install and use nvm for node version compatibility
wget -qO- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.6/install.sh | bash
nvm use lts/hydrogen  # node >= 18

# 2 - Package install
yarn add @nibiruchain/nibijs
# NOTE: You should use v1 or higher for mainnet. For network API guarantees, the
# major and minor version of nibijs should version of the chain you're using.
# I.e., 1.5.x and 1.5.y should be compatible.
yarn install --check-files
```

## Nibiru JS (`nibijs`): Query and Tx Clients

```js
import {
  NibiruTxClient,
  NibiruQuerier,
  Chain,
  Testnet,
  newRandomWallet,
} from "@nibiruchain/nibijs"

/**
 * A "Chain" object exposes all endpoints for Nibiru node such as the
 * gRPC server, Tendermint RPC endpoint, and REST server.
 *
 * The most important one for nibijs is "Chain.endptTM", the Tendermint RPC.
 **/
const chain: Chain = Testnet() // Permanent testnet

// ---------------- NibiruQuerier   ----------------
const querier = await NibiruQuerier.connect(chain.endptTm)

// ---------------- NibiruTxClient ----------------
// let signer = await newRandomWallet() // Signer: randomly generated
let signer = await newSignerFromMnemonic("mnemonic here...") // Signer: in-practice
const txClient = await NibiruTxClient.connectWithSigner(
  CHAIN.endptTm,
  signer
)
```

## Nibiru JS (`nibijs`): Common Queries

Once you have the `NibiruQuerier` (called `querier`) connected, you can use the following:

| Behavior | Description |
| ---  | --- |
| `querier.getAllBalances(address)` | Return all token balances for the given address |
| `querier.getBalance(address, denom)` | Return a single token balance for the given address |
| `querier.tm.block(height)` | Return block details by height such as the block has, parent block hash, block height, timestamp, etc. |
| `querier.getHeight()` | Return latest block height |
| `querier.getBlock()` | Return latest block. Passing in the height as an argument is equivalent to `querier.tm.block(height)`. |
| `querier.getTxByHash(txHashHex)` | Request transaction details using the hexadecimal string for the transaction hash. |
| `querier.getTxByHashSha(txHashSha)` | Request transaction details using its SHA-256 hash, endoed as an array of bytes (`Uint8Array` in TS, `Vec<u8>` in Rust). |

#### Transaction Hashes

The `txHash` returned in block results is  a hexadecimal-encoded version of the
SHA-256 cryptographic hash. If you have the tx hash in SHA-256 / bytes form, use
`getTxByHashSha`. And if you have the hex string (more common form), use
`getTxByHash`.

#### BlockResponse.block.header from `querier.tm.block(height)`

```js
export interface Header {
    readonly version: Version;
    readonly chainId: string;
    readonly height: number;
    readonly time: ReadonlyDateWithNanoseconds;
    /** Block ID of the previous block. This is only `null` for block height 1. */
    readonly lastBlockId: BlockId | null;
    /** Hashes of block data. */
    readonly lastCommitHash: Uint8Array;
    readonly dataHash: Uint8Array;
    readonly validatorsHash: Uint8Array;
    readonly nextValidatorsHash: Uint8Array;
    readonly consensusHash: Uint8Array;
    /** This can be an empty string for height 1 and turn into "0000000000000000" later on 🤷 */
    readonly appHash: Uint8Array;
    /** This is `sha256("")` when there no data */
    readonly lastResultsHash: Uint8Array;
    /** This is `sha256("")` when there no data */
    readonly evidenceHash: Uint8Array;
    readonly proposerAddress: Uint8Array;
}
```

## Wallets

- [Wallets | Nibiru Docs](../../wallets)

## Test Network Tokens (Faucet)

Web Faucet: [app.nibiru.fi/faucet](https://app.nibiru.fi/faucet)

## Running a Full Node

Please note that for mainnet, you’ll need the latest [release from NiibiruChain/nibiru](https://github.com/NibiruChain/nibiru/releases) rather than the version used on testnet.

- [Networks of Nibiru](https://nibiru.fi/docs/dev/networks/) (API Endpoints)
- [Joining Testnet with a Full Node](https://nibiru.fi/docs/run-nodes/testnet/)
