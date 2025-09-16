# 📘 Glossary

A specialized reference page with definitions and explanations for concepts and
jargon related to Nibiru Chain. {synopsis}

<!-- omit in toc -->

- [📘 Glossary](#-glossary)
  - [Glossary — General](#glossary--general)
    - [Block](#block)
    - [Blockchain](#blockchain)
    - [Cosmos Hub / Gaia](#cosmos-hub--gaia)
    - [Cosmos-SDK](#cosmos-sdk)
    - [IBC Relayer](#ibc-relayer)
    - [Proof of Stake (PoS)](#proof-of-stake-pos)
    - [Tendermint Core \& the Application Blockchain Interface (ABCI)](#tendermint-core--the-application-blockchain-interface-abci)
  - [Glossary — Nibi-Swap](#glossary--nibi-swap)
    - [Automated Market-Maker (AMM)](#automated-market-maker-amm)
    - [Swap](#swap)
    - [Liquidity](#liquidity)
    - [Liquidity Provider](#liquidity-provider)
    - [Total Value Locked (TVL)](#total-value-locked-tvl)
  - [Glossary — Nibi-Perps](#glossary--nibi-perps)
    - [Perpetual Swap (Contract)](#perpetual-swap-contract)
    - [Funding Payment](#funding-payment)
  - [References](#references)

***

## Glossary — General

### Block

Groups of information stored on a [blockchain](GLOSSARY.md#blockchain). Each block contains transactions that are grouped, verified, and signed by validators.

### Blockchain

An immutable ledger of transactions maintained across a network of independent computer systems.

### Cosmos Hub / Gaia

The Cosmos Hub, also called Gaia, is the first blockchain created within the Cosmos Ecosystem. Gaia is a proof-of-stake chain supported by the ATOM token, which is used for staking, governance, and block rewards. Gaia is also one of the main relayers of data between blockchains in the Cosmos Ecosystem.

### Cosmos-SDK

An open-source framework for building multi-asset public blockchains like [Gaia](https://hub.cosmos.network/), [Binance Chain](https://docs.binance.org/), and [Juno](https://docs.junonetwork.io/juno/readme). Nibiru is built on the Cosmos-SDK. Check out the [Cosmos-SDK docs](https://docs.cosmos.network/main/intro/overview.html) for more information.

### IBC Relayer

In the Inter-Blockchain Communication (IBC) Protocol, blockchains do not directly pass messages to each other over the network. Instead, relayers monitor for updates on open paths between sets of IBC enabled chains and then submits their own updates with specific message types to the counterparty chains. Clients are then used to track and verify the consensus state. For more information on IBC concepts like relayers, connections, packets, clients, and channels, visit [ibc.cosmos.network](https://ibc.cosmos.network/).

### Proof of Stake (PoS)

Proof-of-stake is a type of consensus mechanism used by blockchains to achieve distributed consensus. In proof-of-work, miners prove they have capital at risk by expending energy. In Nibiru proof-of-stake, validators explicitly stake capital in the form of NIBI. Staked NIBI acts as collateral that can be destroyed if the validator behaves dishonestly or lazily. Validators are responsible for creating new blocks, propagating blocks across the network, and verifying that blocks are valid. PoS is also

### Tendermint Core & the Application Blockchain Interface (ABCI)

**Tendermint Consensus** consists of two chief technical components: a blockchain consensus
engine and a generic application interface.

| | |
| --- | --- |
| [Tendermint Core](https://docs.tendermint.com/) |  The consensus engine, called [Tendermint Core](https://docs.tendermint.com/), ensures that the same transactions are recorded on every machine in the same order. |
| [Application Blockchain Interface (ABCI)](https://docs.tendermint.com/master/spec/abci/) | The application interface, called the [Application Blockchain Interface (ABCI)](https://docs.tendermint.com/master/spec/abci/), enables the transactions to be processed in any programming language. |

Tendermint has evolved to be a general purpose blockchain consensus engine that
can host arbitrary application states. Since Tendermint can replicate arbitrary
applications, it can be used as a plug-and-play replacement for the consensus
engines of other blockchains. Evmos is such an example of an ABCI application
replacing Ethereum's PoW via Tendermint's consensus engine.

Another example of a cryptocurrency application built on Tendermint is the Cosmos
network. Tendermint is able to decompose the blockchain design by offering a very
simple API (ie. the ABCI) between the application process and consensus process.

Tendermint Byzantine Fault Tolerant (BFT) consensus is a solution that packages the networking and consensus layers of a blockchain into a generic engine, allowing developers to focus on application development as opposed to the complex underlying protocol. Tendermint provides the equivalent of a web-server, database, and supporting libraries for blockchain applications written in any programming language.

***

## Glossary — Nibi-Swap

### Automated Market-Maker (AMM)

Automated market makers are mechanisms in the decentralized finance (DeFi) ecosystem that allow digital assets to be traded in a permissionless and automatic way using liquidity pools rather than a traditional market of buyers and sellers. AMM users supply liquidity pools with crypto tokens. Token prices in a pool are determined by a fixed mathematical formula.

### Swap

Trading one cryptocurrency for another.

### Liquidity

Digital assets stored in a pool that are able to be traded against are liquid. These assets are often just called liquidity.

### Liquidity Provider

A liquidity provider is someone that deposits assets into a liquidity pool.

### Total Value Locked (TVL)

TVL is the overall value of crypto assets deposited into some structure (often denoted in US dollars). Usually, TVL is given for pools or entire protocols.

***

## Glossary — Nibi-Perps

### Perpetual Swap (Contract)

Perpetual swap contracts are [swaps](GLOSSARY.md#swap) that allow traders to buy or sell a derivative of some underlying asset without expiration and with a potentially

### Funding Payment

Funding payments are regularly scheduled payments between longs and shorts on Nibi-Perps. These payments are meant to converge the price between the derivative contract (mark), or perp, and its underlying asset (index). A time-weighted average price from the virtual pool is taken to compute the mark price. The index price is derived from an oracle. Funding payments are calculated and exchanged between traders every half hour on Nibiru.

***

## References

- [Tendermint Core Docs](https://docs.tendermint.com)
- [Cosmos-SDK Docs](https://docs.cosmos.network)
- [BitMEX. Perpetual Contracts Guide](https://www.bitmex.com/app/perpetualContractsGuide)
- [Inter-Blockchain Communication (IBC) Docs](https://ibc.cosmos.network/)
