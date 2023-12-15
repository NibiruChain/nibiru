# Nibiru Chain

[![Go Reference](https://pkg.go.dev/badge/github.com/NibiruChain/nibiru.svg)](https://pkg.go.dev/github.com/NibiruChain/nibiru)
[![Nibiru Test workflow][badge-go-linter]][workflow-go-linter]
[![Nibiru Test workflow][badge-go-releaser]][workflow-go-releaser]
[![GitHub][license-badge]](https://github.com/NibiruChain/nibiru/blob/master/LICENSE.md)
[![Discord Badge](https://dcbadge.vercel.app/api/server/nibirufi?style=flat)](https://discord.gg/nibirufi)

**Nibiru Chain** is a proof-of-stake blockchain that unifies leveraged derivatives trading, spot trading, staking, and bonded liquidity provision into a seamless user experience, enabling users of over 40 blockchains to trade with leverage using a suite of composable decentralized applications.

## Components of Nibiru

- **CosmWasm Smart Contracts**: Rust-based, WebAssembly (WASM) smart contracts built for the Cosmos Ecosystem. See our [CosmWasm sandbox monorepo (cw-nibiru)](https://github.com/NibiruChain/cw-nibiru/tree/main) for the protocol's core smart contracts. 
- **Nibi-Perps**: A perpetual futures exchange where users can take leveraged exposure and trade on a plethora of assets — completely on-chain, completely non-custodially, and with minimal gas fees.
- **Oracle Module**: Nibiru accurately prices assets using a native, system of decentralized oracles, and communicates with other Cosmos layer-1 chains using the Inter-Blockchain Communication (IBC) (opens new window)protocol.
- **Nibi-Swap**: An automated market maker protocol for multichain assets. This application gives users access to swaps, pools, and bonded liquidity gauges.

## Modules

| Module |  Description |
| --- | --- | 
| [common][code-x-common] | Holds helper and utility functions to be utilized by other `x/` modules. |
| [epochs][code-x-epochs] | Often in the SDK, we would like to run certain code every-so often. The purpose of `epochs` module is to allow other modules to set that they would like to be signaled once every period. So another module can specify it wants to execute code once a week, starting at UTC-time = x. `epochs` creates a generalized epoch interface to other modules so that they can easily be signalled upon such events. |
| [inflation][code-x-inflation] | Implements the [tokenomics](https://nibiru.fi/docs/learn/tokenomics.html) for Nibiru. |
| [oracle][code-x-oracle] | Handles the posting of an up-to-date and accurate feed of exchange rates from the validators. | 
| [perp][code-x-perp] | Powers the Nibi-Perps exchange. This module enables traders to open long and short leveraged positions and houses all of the PnL calculation and liquidation logic. |
| [spot][code-x-spot] | Responsible for creating, joining, and exiting liquidity pools. It also allows users to swap between two assets in an existing pool. It's a fully functional AMM. |
| [wasm][code-x-wasm] | Implements the execution environment for [WebAssembly (WASM) smart contracts](https://book.cosmwasm.com/). |

[code-x-common]: https://github.com/NibiruChain/nibiru/tree/master/x/common
[code-x-epochs]: https://github.com/NibiruChain/nibiru/tree/master/x/epochs
[code-x-inflation]: https://github.com/NibiruChain/nibiru/tree/master/x/inflation
[code-x-oracle]: https://github.com/NibiruChain/nibiru/tree/master/x/oracle
[code-x-perp]: https://github.com/NibiruChain/nibiru/tree/master/x/perp
[code-x-spot]: https://github.com/NibiruChain/nibiru/tree/master/x/spot
[code-x-wasm]: https://github.com/NibiruChain/nibiru/tree/main/wasmbinding

Nibiru is built with the [Cosmos-SDK][cosmos-sdk-repo] on [Tendermint Core](https://tendermint.com/core/) consensus, accurately prices assets using a system of decentralized oracles, and communicates with other Cosmos layer-1 chains using the [Inter-Blockchain Communication (IBC)](https://github.com/cosmos/ibc) protocol.  

## ⚙️ — Documentation

Conceptual and technical documentation can be found in the [Nibiru docs](https://docs.nibiru.fi). Detailed module-specific documentation is included in the top-level README (`x/module/README.md)`.

## 💬 — Community

If you have questions or concerns, feel free to connect with a developer or community member in the [Nibiru discord][social-discord]. We also have active communities on Twitter and Telegram.

<!-- Markdown versions of the social badges 
[![description][discord-badge]][social-discord] 
[![description][twitter-badge]][social-twitter] 
[![description][telegram-badge]][social-telegram]
-->

<p style="display: flex; gap: 24px; justify-content: center; text-align:center">
<a href="https://discord.gg/nibiruchain"><img src="https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white" alt="Discord" height="22"/></a>
<a href="https://twitter.com/NibiruChain"><img src="https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white" alt="Tweet" height="22"/></a>
<a href="https://t.me/nibiruhackathon"><img src="https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white" alt="Telegram" height="22"/></a>
</p>

----

## ⛓️ Installation: Developing on the chain locally

Installation instructions for the `nibid` binary can be found in [INSTALL.md](./INSTALL.md).

Recommended minimum specs:

- 2CPU, 4GB RAM, 100GB SSD
- Unix system: MacOS or Ubuntu 18+

## Nibid CLI

To simply access the `nibid` CLI, run:

```bash
make install
```

Usage instructions for the `nibid` CLI are available at [docs.nibiru.fi/dev/cli](https://docs.nibiru.fi/dev/cli/) and the [Nibiru Module Reference](https://docs.nibiru.fi/dev/x/).

## Running a Local Node

On a fresh clone of the repo, simply run:
```bash
make localnet
``` 
and open another terminal.  

## Generate the protobufs

```bash
make proto-gen
```

## Linter

We use the [golangci-lint](https://golangci-lint.run/) linter. Install it and run

```sh
golangci-lint run
```

at the root directory. You can also install the VSCode or Goland IDE plugins.

## Multiple Nodes

Run the following commands to set up a local network of Docker containers running the chain.

```sh
make build-docker-nibidnode

make localnet-start
```

## License

Licensed under the [MIT License](./LICENSE.md).

[license-badge]: https://img.shields.io/badge/License-MIT-blue.svg
[cosmos-sdk-repo]: https://github.com/cosmos/cosmos-sdk
[badge-go-linter]: https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml/badge.svg?query=branch%3Amaster
[workflow-go-linter]: https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml?query=branch%3Amaster
[badge-go-releaser]: https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml/badge.svg?query=branch%3Amaster
[workflow-go-releaser]: https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml?query=branch%3Amaster

[social-twitter]: https://twitter.com/NibiruChain
[social-discord]: https://discord.gg/nibirufi
[social-telegram]: https://t.me/nibiruchain

[discord-badge]: https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white
[twitter-badge]: https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white
[telegram-badge]: https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white

<!--
[![Twitter Follow](https://img.shields.io/twitter/follow/nibiru_platform.svg?label=Follow&style=social)][social-twitter]

[![version](https://img.shields.io/github/tag/nibiru-labs/nibiru.svg)](https://github.com/NibiruChain/nibiru/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/NibiruChain/nibiru)](https://goreportcard.com/report/github.com/NibiruChain/nibiru) 

[![API Reference](https://godoc.org/github.com/NibiruChain/nibiru?status.svg)](https://godoc.org/github.com/NibiruChain/nibiru)

[![Discord Chat](https://img.shields.io/discord/704389840614981673.svg)][social-discord]
-->
