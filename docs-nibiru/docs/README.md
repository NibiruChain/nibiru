---
title: Nibiru Docs / Whitepaper
canonicalUrl: "https://nibiru.fi/docs/"
description: "Whitepaper and product documentation for Nibiru, a blazingly fast blockchain built to support real-time applications and scale to millions of Web3 users. Nibiru powers a smart contract hub with RWAs, DeFi, and more."
---

# Nibiru

Nibiru is a breakthrough L1 blockchain and smart contract ecosystem
providing superior throughput and unparalleled security. Nibiru  aims to be the
most **developer-friendly and user-friendly** smart contract platform in Web3. 

<!--
"If there are two evils in this world, they're centralization and complexity." — George Hotz
-->

<template>
  <HeroBoxes :boxes="boxesMain" />
</template>

<script>
const boxesMain = [
  { id: 1, title: "Build on Nibiru", href: "/docs/dev/",
    text: `From smart contract guides to essential Web3 primitives and infrastructure, utilize Nibiru's developer friendly SDKs to build decentralized applications.`},
  { id: 2, title: "Use Nibiru Chain", href: "/docs/use/",
    text: `Kickstart your journey as a user of Nibiru. Learn how to create wallets, stake NIBI, participate in campaigns, or use the Nibiru web application.` },
  { id: 3, title: "Learn Core Concepts", href: "/docs/concepts/",
    text: `Build an understanding of the core concepts behind how Nibiru Chain works. Topics explored in this section provide a solid foundation on both the Nibiru blockchain and Web3 in general.` },
  { id: 4, title: "Join the Community", href: "/docs/community/",
    text: `Engage with the Nibiru community, a global community of software developers, content creators, node runners, and Web3 enthusiasts.`,
  },
]
const boxesEnd = [
  { id: 1, title: "NibiruBFT: Consenus Engine", href: "/docs/dev/",
    text: `Instant finality, consistent security, and Byzantine Fault Tolerant state machine replication.`},
  { id: 2, title: "Frequently Asked Questions (FAQ)", href: "/docs/learn/faq/",
    text: `Kickstart your journey as a user of Nibiru. Learn how to create wallets, stake NIBI, participate in campaigns, or use the Nibiru web application.` },
  { id: 3, title: "Nibiru Wasm", href: "/docs/ecosystem/wasm/",
    text: `WebAssembly (Wasm) smart contract execution environment programmed in Rust.` },
  { id: 4, title: "Nibiru Roadmap (2025)", href: "/docs/ecosystem/future/",
    text: `A brief sneak peek at onging and upcoming improvements to Nibiru.`,
  },
  { id: 5, title: "Tokenomics", href: "/docs/learn/tokenomics.html",
    text: `Understand NIBI, the staking and utility token of Nibiru Chain that
powers the network's consensus engine, governance, and computation as "gas".`,
  },
  { id: 6, title: "Guide: Building with NibiJS", href: "/docs/dev/tools/nibijs/",
    text: `A brief guide on Nibiru's APIs, docs, and resources to broadcast transactions and query the chain.`,
  },
]
const boxesUsers = [
  { id: 1, title: "Community Hub", href: "/docs/community/",
    text: ``},
  { id: 2, title: "Nibiru Web App", href: "https://app.nibiru.fi/stake",
    text: `` },
  { id: 3, title: "Guide: Set up a Nibiru Wallet", href: "/docs/wallets/",
    text: `` },
  { id: 4, title: "Guide: Staking on Nibiru", href: "/docs/use/stake.html",
    text: ``,
  },
]
const boxesDevs = [
  { id: 1, title: "Developer Hub - Build on Nibiru", href: "/docs/dev/",
    text: `Everything you need to build on Nibiru. Your go-to hub to develop smart contracts applications for the decentralized web.`},
]
export default {
  data() {
    return {
      boxesMain,
      boxesEnd,
      boxesUsers,
      boxesDevs,
    }
  }
}
</script>

## For Users

Engage with Nibiru's fast-growing community or get started by accessing a wealth of resources and tutorials below.

- [Nibiru Community Hub](./community/)
- [Nibiru Web App](https://app.nibiru.fi/)
- [Guide: Set Up a Nibiru Chain Wallet](./wallets/)
- [Guide: Staking on Nibiru](./use/stake.html)

## For Devs

<template>
  <HeroBoxes :boxes="boxesDevs" />
</template>

- [Smart Contract Sandbox (NibiruChain/nibiru-wasm)](https://github.com/NibiruChain/nibiru-wasm/tree/main)
- [TypeScript SDK: NibiJS](./dev/tools/kickstart.html)
- [Rust SDK: `nibiru_std`](https://github.com/NibiruChain/nibiru-wasm/tree/main/contracts#example-contracts)
- [Golang SDK: Gonibi](./dev/tools/go-sdk.html)
- [Python SDK](./dev/tools/py-sdk.html)

<a href="./dev/">
<img src="./img/chain-arch.png" style="border-radius: 16px;">
</a>

## Learn About Nibiru

Nibiru acts as a permission-less platform for developers to deploy secure,
production-grade smart contracts.

- [Smart Contracts on Nibiru](./ecosystem/wasm)
- [Learn: Core Concepts](./concepts/tx-msgs.html)
- [Learn: Blockchain Modules](./arch/)

<template>
  <HeroBoxes :boxes="boxesEnd" />
</template>

<a href="./ecosystem/future/">
<img style="border-radius: 1.5rem;" src="./img/2024-roadmap-nibiru.png">
</a>

## Nibiru Ecosystem: Featured Apps

::: tip
Explore a more comprehensive set of projects building on Nibiru in our [Ecosystem Hub](https://nibiru.fi/ecosystem).
:::

- [**Sai.fun Perps Exchange**](./ecosystem/apps/sai-fun-perps.md): A perpetual futures exchange where
  users can take leveraged exposure and trade on a plethora of assets —
  completely on-chain, completely non-custodially, and with minimal gas fees.

- [**Astrovault**](https://nibiru.fi/ecosystem/apps/astrovault): A unique
exchange prioritizing efficiency, low-friction trading, and rewards.

- [**Nibiru Oracle**](./ecosystem/oracle/index.md): Nibiru accurately prices assets
  using a native, system of decentralized oracles. Both external APIs and smart
  contracts can tap into the Oracle Module for secure, low-latency feeds.

<!-- 
- [**Coded Estate**](https://codedestate.com/):  Coded Estate is about bringing
  homes on chain, rentals on chain, and democratizing access into the real estate
  system.  Coded Estate is reimagining ownership, decentralized allowing
  ownership for any and everybody.
-->

---

<!--TODO Extract content from below SVGs for use on dedicated page. -->
<!-- ![](./img/cosmwasm-ibc-box.svg) -->
<!-- ![](./img/cosmos-sdk-tendermint-box.svg) -->

<!--TODO Write and add link to local IBC page-->
<!-- - [Inter-Blockchain Communication (IBC)](https://ibc.cosmos.network/):  Nibiru communicates -->
<!--   with other Cosmos layer-1 chains using the IBC protocol. IBC enables secure -->
<!--   and censorship-resistant transfers of funds between blockchains, cross-chain -->
<!--   computation, and transmission of arbitrary data. -->

<!--
Ref: https://github.com/cosmos/ibc

This includes cross-chain smart contract calls, fee payments, NFTs, and
fungible token transfers. IBC is not reliant on a multi-sig or centralized
bridging solution.
-->

<!--
The security of the Nibiru blockchain relies on a set of validators to commit
new blocks and participate in Tendermint BFT consensus by brodcasting votes
that contain cryptographic signatures signed by each validator's private key.
Validators stake **NIBI**, the protocol's native token used for gas,
governance, and "mining". Users can delegate NIBI to validators that record and
verify transactions in exchange for rewards.
-->


## Contribution Guidelines

You can contribute to improve this documentation by submitting a
GitHub ticket in our [`website-help` repository](https://github.com/NibiruChain/website-help).
