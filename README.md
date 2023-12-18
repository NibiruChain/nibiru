# Nibiru Chain

[![Go Reference](https://pkg.go.dev/badge/github.com/NibiruChain/nibiru.svg)](https://pkg.go.dev/github.com/NibiruChain/nibiru)
[![Nibiru Test workflow][badge-go-linter]][workflow-go-linter]
[![Nibiru Test workflow][badge-go-releaser]][workflow-go-releaser]
[![GitHub][license-badge]](https://github.com/NibiruChain/nibiru/blob/main/LICENSE.md)
[![Discord Badge](https://dcbadge.vercel.app/api/server/nibirufi?style=flat)](https://discord.gg/nibirufi)

**Nibiru Chain** is a breakthrough smart contract platform providing superior throughput, reduced latency, and improved security, all driven by Web Assembly (Wasm) smart contracts.

- [⚙️ — Documentation](#%EF%B8%8F--documentation)
- [💬 — Community](#--community)
- [🧱 — Components of Nibiru](#--components-of-nibiru)
- [⛓️ — Building: `make` commands](#%EF%B8%8F--building-make-commands)
  - [Nibid CLI](#nibid-cli)
  - [Running a Local Node](#running-a-local-node)
  - [Generate the protobufs](#generate-the-protobufs)
  - [Linter](#linter)
  - [Multiple Nodes](#multiple-nodes)
- [License](#license)

## ⚙️ — Documentation

- [Docs | Nibiru Chain](https://nibiru.fi/docs/): Conceptual and technical documentation can be found here. 
- [Complete Golang reference docs](https://pkg.go.dev/github.com/NibiruChain/nibiru): (`pkg.go.dev`) For the blockchain implementation .
- ["X Module"](https://nibiru.fi/docs/dev/x/)-specific documentation

## 💬 — Community

If you have questions or concerns, feel free to connect with a developer or community member in the [Nibiru Discord][social-discord]. We also have active communities on [Twitter][social-twitter] and [Telegram][social-telegram].

<!-- Markdown versions of the social badges 
[![description][discord-badge]][social-discord] 
[![description][twitter-badge]][social-twitter] 
[![description][telegram-badge]][social-telegram]
-->

<p style="display: flex; gap: 24px; justify-content: center; text-align:center">
<a href="https://discord.gg/nibiruchain"><img src="https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white" alt="Discord" height="22"/></a>
<a href="https://twitter.com/NibiruChain"><img src="https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white" alt="Tweet" height="22"/></a>
<a href="https://t.me/nibiruchain"><img src="https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white" alt="Telegram" height="22"/></a>
</p>

## 🧱 — Components of Nibiru

- **CosmWasm Smart Contracts**: Rust-based, Wasm smart contracts built enhanced security, performance, and interoperability. See our [CosmWasm sandbox monorepo (cw-nibiru)](https://github.com/NibiruChain/cw-nibiru/tree/main) for the protocol's core smart contracts. 
- **Nibi-Perps**: A perpetual futures exchange where users can take leveraged exposure and trade on a plethora of assets — completely on-chain, completely non-custodially, and with minimal gas fees.
- **Nibi-Oracle**: Nibiru accurately prices assets using a native, system of decentralized oracles, and communicates with other Cosmos layer-1 chains using the Inter-Blockchain Communication (IBC) (opens new window)protocol.
- **Nibi-Swap**: An automated market maker protocol for multichain assets. This application gives users access to swaps, pools, and bonded liquidity gauges.

| Module |  Description |
| --- | --- | 
| [wasm][code-x-wasm] | Implements the execution environment for [WebAssembly (WASM) smart contracts](https://nibiru.fi/docs/wasm/). |
| [devgas][code-x-devgas] | The `devgas` module of Nibiru Chain shares contract execution fees with smart contract developers. This aims to increase the adoption of Nibiru by offering CosmWasm smart contract developers a direct source of income based on usage. |
| [epochs][code-x-epochs] | The `epochs` module allows other modules to set hooks to be called to execute code automatically on a period basis. For example, "once a week, starting at UTC-time = x". `epochs` creates a generalized epoch interface. |
| [inflation][code-x-inflation] | Implements the [tokenomics](https://nibiru.fi/docs/learn/tokenomics.html) for Nibiru. |
| [common][code-x-common] | Holds helper and utility functions to be utilized by other `x/` modules. |
| [oracle][code-x-oracle] | Handles the posting of an up-to-date and accurate feed of exchange rates from the validators. | 
| [perp][code-x-perp] | Powers the Nibi-Perps exchange. This module enables traders to open long and short leveraged positions and houses all of the PnL calculation and liquidation logic. |
| [spot][code-x-spot] | Responsible for creating, joining, and exiting liquidity pools. It also allows users to swap between two assets in an existing pool. It's a fully functional AMM. |

[code-x-common]: https://github.com/NibiruChain/nibiru/tree/main/x/common
[code-x-devgas]: https://nibiru.fi/docs/dev/x/nibiru-chain/devgas.html
[code-x-epochs]: https://github.com/NibiruChain/nibiru/tree/main/x/epochs
[code-x-inflation]: https://github.com/NibiruChain/nibiru/tree/main/x/inflation
[code-x-oracle]: https://github.com/NibiruChain/nibiru/tree/main/x/oracle
[code-x-perp]: https://github.com/NibiruChain/nibiru/tree/main/x/perp
[code-x-spot]: https://github.com/NibiruChain/nibiru/tree/main/x/spot
[code-x-wasm]: https://github.com/NibiruChain/nibiru/tree/main/wasmbinding

Nibiru is built with the [Cosmos-SDK][cosmos-sdk-repo] on [Tendermint Core](https://tendermint.com/core/) consensus and communicates with other blockchain chains using the [Inter-Blockchain Communication (IBC)](https://github.com/cosmos/ibc) protocol.

----

## ⛓️ — Building: `make` commands

Installation instructions for the `nibid` binary can be found in [INSTALL.md](./INSTALL.md).

Recommended minimum specs:

- 2CPU, 4GB RAM, 100GB SSD
- Unix system: MacOS or Ubuntu 18+

### Nibid CLI

To simply access the `nibid` CLI, run:

```bash
make install
```

Usage instructions for the `nibid` CLI are available at [docs.nibiru.fi/dev/cli](https://docs.nibiru.fi/dev/cli/) and the [Nibiru Module Reference](https://docs.nibiru.fi/dev/x/).

### Running a Local Node

On a fresh clone of the repo, simply run:
```bash
make localnet
``` 
and open another terminal.  

### Generate the protobufs

```bash
make proto-gen
```

### Linter

We use the [golangci-lint](https://golangci-lint.run/) linter. Install it and run

```sh
golangci-lint run
```

at the root directory. You can also install the VSCode or Goland IDE plugins.

### Multiple Nodes

Run the following commands to set up a local network of Docker containers running the chain.

```sh
make build-docker-nibidnode

make localnet-start
```

## License

Licensed under the [MIT License](./LICENSE.md).

[license-badge]: https://img.shields.io/badge/License-MIT-blue.svg
[cosmos-sdk-repo]: https://github.com/cosmos/cosmos-sdk
[badge-go-linter]:
https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml/badge.svg?query=branch%3Amain
[workflow-go-linter]:
https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml?query=branch%3Amain
[badge-go-releaser]:
https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml/badge.svg?query=branch%3Amain
[workflow-go-releaser]:
https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml?query=branch%3Amain

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
