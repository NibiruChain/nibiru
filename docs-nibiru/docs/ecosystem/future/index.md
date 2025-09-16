---
order: 4
---

# Nibiru Roadmap (2025)

A brief sneak peek at onging and upcoming improvements to Nibiru. {synopsis}

<img style="border-radius: 1.5rem;" src="../../img/2025-roadmap-nibiru.png">

**Table of Contents**

- [EVM Support](#evm-support)
- [IBC Middleware](#ibc-middleware)
- [Interchain Queries and Accounts: ICA, ICQ, and Hooks](#interchain-queries-and-accounts-ica-icq-and-hooks)

## EVM Support

For developers, the ability to build and deploy applications in a familiar EVM
environment while enjoying the benefits of improved scalability, faster time to
finality, interblockchain composability (IBC), and modularity with Wasm smart
contracts—all with a lower barrier to entry—is a major plus.

EVM public mainnet is now live (2025).  Nibiru enables full EVM
compatibility by adapting the Ethereum protocol logic from Geth ([Go-Ethereum](https://github.com/ethereum/go-ethereum/blob/b946b7a13b749c99979e312c83dce34cac8dd7b1/core/types/transaction.go#L43-L48)) to make Nibiru's full nodes operate as execution environments for both Wasm and the EVM simultaneously.

<!-- In this way, Nibiru helps alleviate what's commonly referred to as "the -->
<!-- blockchain trilemma", a challenging problem to obviate the tradeoffs between -->
<!-- addressing decentralization, security, and scalability concurrently. -->

<!-- [Ethermint](https://github.com/evmos/ethermint/tree/07cf2bd2b1ce9bdb2e44ec42a39e7239292a14af/x) -->
<!-- to be compatible with ABCI++ (CometBFT v0.37). -->

<!--
| Topic | EVM rollup | Native EVM |
| ----- | ---------- | ---------- |
| Depencency | EVM rollup | Native EVM |
| Code surface area | EVM rollup | Native EVM |
| Throughput | EVM rollup | Native EVM |
| Implementation complexity | EVM rollup | Native EVM |

To explore: Latency
   - Security: A native module introduces more surface area for potential vulnerabilities.
-->

## IBC Middleware

IBC apps function as standalone modules, each possessing their own specialized
logic. These modules interface with the core IBC handlers that maintain the
integrity of IBC's foundational properties: transport, authentication, and
ordering. Rather than delving into the intricacies of each application, these
handlers enable the IBC app modules to oversee those nuances.

Still, there arise situations where common functionalities are sought by
multiple applications but aren't fit for integration into core IBC.

This is where **IBC middleware** comes into play. It provides developers the
flexibility to craft extensions as distinct modules that can overlay the base
app. The middleware can implement its distinct logic and transfer data to the
app (module), which remains unaware of the middleware's actions.

Such a design permits the introduction of tailored functionalities without
modifying the core IBC or the base apps of various chains. It paves the way for
both pre and post-processing, allowing the app modules to focus on
specialization. The modular nature of the IBC middleware encourages code reuse
across different IBC applications.

IBC middleware offers a non-intrusive method to enhance IBC app modules,
establishing a distinct separation of duties between the core IBC, middleware,
and app logic. This setup ensures that both the main app and middleware
maintain their independent logic, yet collaborate effectively within a unified
packet flow.

#### IBC Middleware Use Cases

1. Fee middleware: Create economies around IBC infra by including fees for packet processing.
2. Packet-Forwarding Middleware (PMF): Add custom logic on processing of IBC packets.

#### Related Reading

1. [Strangelove-ventures/packet-forward-middle](https://github.com/strangelove-ventures/packet-forward-middleware)
2. [Cosmos/ibc-apps](https://github.com/cosmos/ibc-apps)
3. [IBC-Go: IBC Middleware](https://ibc.cosmos.network/main/ibc/middleware/overview.html)


## Interchain Queries and Accounts: ICA, ICQ, and Hooks

#### Interchain Accounts (ICA)

- **ICA Destination**: This feature allows the creation of accounts that exist across multiple blockchains, called interchain accounts (ICAs).
  - Provides cross-chain account abstraction. Users will be able to hold fungible tokens from different chains like Gaia, Axelar, and Osmosis from one account.
  - Abstractscross-chain asset transfers. ICAs appear as regular accounts on each blockchain.
  - Handles asset locking, unlocking, and proofs.
- **ICA Controller**: Management of ICAs across chains.
  - Govern an ICA by setting deposit/withdraw limits, allowed transaction types, etc.
  - Can pause, resume, or close an ICA as needed.
  - Enables central management of ICAs across many heterogeneous chains.

#### IBC Hooks

The IBC hooks module is an IBC middleware that enables ICS-20 token transfers to initiate CosmWasm smart contract calls. This opens up many possibilies for extensibility.


