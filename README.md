# Nibiru

[![Go Reference](https://pkg.go.dev/badge/github.com/NibiruChain/nibiru.svg)](https://pkg.go.dev/github.com/NibiruChain/nibiru/v2#section-readme)
[![Nibiru Test workflow][badge-go-linter]][workflow-go-linter]
[![Nibiru Test workflow][badge-go-releaser]][workflow-go-releaser]
[![GitHub][license-badge]](https://github.com/NibiruChain/nibiru/blob/main/LICENSE.md)

**Nibiru** is a breakthrough Layer 1 blockchain and smart contract ecosystem providing superior throughput, improved security, and a high-performance EVM execution layer. Nibiru aims to be the most developer-friendly and user-friendly smart contract ecosystem, leading the charge toward mainstream Web3 adoption by innovating at each layer of the stack: dApp development, scalable blockchain data indexing, consensus optimizations, a comprehensive developer toolkit, and composability across multiple VMs.

**Table of Contents**:

- [Documentation](#documentation)
- [Community](#community)
- [Nibiru Core Architecture](#nibiru-core-architecture)
  - [Execution Extensions](#execution-extensions)
  - [Auxiliary Modules](#auxiliary-modules)
  - [Consensus Engine](#consensus-engine)
- [Developing in this Codebase](#developing-in-this-codebase)
  - [Docker Development Environment](#docker-development-environment)
- [How to Install Nibiru](#how-to-install-nibiru)
- [License](#license)

---

## Documentation

- [Nibiru Docs (nibiru.fi/docs)](https://nibiru.fi/docs/): This is the best resource for comprehensive technical, conceptual, and product documentation on Nibiru.
- [Complete Golang reference docs](https://pkg.go.dev/github.com/NibiruChain/nibiru/v2#section-readme): (`pkg.go.dev`) For the blockchain implementation.
- [Nibiru RPC Endpoints](https://nibiru.fi/docs/dev/networks/)
- [Core Tools and Language Clients](https://nibiru.fi/docs/dev/#core-tools-and-language-clients)

## Community

If you have questions or concerns, feel free to connect with a developer or community member in the [Nibiru Discord][social-discord]. We also have active communities on [X][social-twitter] and [Telegram][social-telegram].

<p style="display: flex; gap: 24px; justify-content: center; text-align:center">
<a href="https://discord.gg/nibirufi"><img src="https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white" alt="Discord" height="22"/></a>
<a href="https://x.com/NibiruChain"><img src="https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white" alt="Tweet" height="22"/></a>
<a href="https://t.me/nibiruchain"><img src="https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white" alt="Telegram" height="22"/></a>
</p>

## Nibiru Core Architecture

### Execution Extensions

These sections of the codebase extend or augment core runtime behavior.

| Module | Description |
| --- | --- |
| [EVM](https://github.com/NibiruChain/nibiru/tree/main/x/evm) | Implements Nibiru EVM, which manages an Ethereum Virtual Machine (EVM) state database and enables the execution of Ethereum smart contracts. Nibiru EVM is an extension of "[geth](https://github.com/ethereum/go-ethereum)" along with "web3" and "eth" JSON-RPC methods. |
| [Wasm][code-x-wasm]           | Implements the execution environment for WebAssembly (WASM) smart contracts. CosmWasm smart contracts are Rust-based, Wasm smart contracts built for enhanced security, performance, and interoperability. See our [CosmWasm sandbox monorepo (nibiru-wasm)](https://github.com/NibiruChain/nibiru-wasm/tree/main) for the protocol's core smart contracts. |
| [Eth][code-x-eth]             | Ethereum integration utilities: EVM JSON-RPC server (HTTP/WebSocket) and APIs (eth/net/web3/debug/txpool), EVM transaction indexer for fast lookups, EIP-155 chain IDs, and EIP-712 signing helpers. See also [server][code-app-server] for JSON-RPC bootstrap and config. |
| [App][code-app]               | Core application logic including custom ante handlers for transaction preprocessing, gas management, signature verification, and EVM integration. Key features include oracle gas optimization, zero-gas actors, and enhanced security guards. |
| [x/nutil][code-x-nutil]       | Helper and utility functions to be utilized by other `x/` modules. |

### Auxiliary Modules

| Module | Description |
| --- | --- |
| [Devgas][code-x-devgas]       | The `devgas` module of Nibiru shares contract execution fees with smart contract developers. This aims to increase the adoption of Nibiru by offering CosmWasm smart contract developers a direct source of income based on usage. |
| [Epochs][code-x-epochs]       | The `epochs` module allows other modules to set hooks to be called to execute code automatically on a period basis. For example, "once a week, starting at UTC-time = x". `epochs` creates a generalized epoch interface.                |
| [Inflation][code-x-inflation] | Implements the [tokenomics](https://nibiru.fi/docs/learn/tokenomics.html) for Nibiru.                                                                                                                                                    |
| [Oracle][code-x-oracle]       | Nibiru accurately prices assets using a native, system of decentralized oracles, and communicates with other Cosmos layer-1 chains using the Inter-Blockchain Communication (IBC) protocol. Nibi-Oracle handles the voting process for validators that act as oracles by updating data feeds.  |
| [Sudo][code-x-sudo]           | Provides an on-chain "root" and a set of whitelisted contracts with elevated permissions. Includes management of Zero Gas Actors for fee-less CosmWasm executions against approved contracts. |

[code-x-nutil]: https://github.com/NibiruChain/nibiru/tree/main/x/nutil
[code-x-devgas]: https://nibiru.fi/docs/dev/x/nibiru-chain/devgas.html
[code-x-epochs]: https://github.com/NibiruChain/nibiru/tree/main/x/epochs
[code-x-inflation]: https://github.com/NibiruChain/nibiru/tree/main/x/inflation
[code-x-oracle]: https://github.com/NibiruChain/nibiru/tree/main/x/oracle
[code-x-wasm]: https://nibiru.fi/docs/wasm/
[code-x-evm]: https://github.com/NibiruChain/nibiru/tree/main/x/evm
[code-x-eth]: https://github.com/NibiruChain/nibiru/tree/main/eth
[code-app-server]: https://github.com/NibiruChain/nibiru/tree/main/app/server
[code-app]: https://github.com/NibiruChain/nibiru/tree/main/app
[code-x-sudo]: https://github.com/NibiruChain/nibiru/tree/main/x/sudo

### Consensus Engine

Nibiru is built on [Tendermint Core (CometBFT)](https://tendermint.com/core/) consensus.


---

## Developing in this Codebase

Install `just` to run project-specific commands.

```bash
cargo install just
```

Nibiru projects use `just` as the command runner instead of `make`. The `just`
tool is a modern command runner that's simpler, more readable, and
self-documenting.

```bash
just            # list all available commands
just build      # build the nibid binary
just localnet   # run a local network
just test-e2e   # run EVM end-to-end tests
```

Ref: [github.com/casey/just](https://github.com/casey/just)

### Docker Development Environment

For a complete local development environment with multiple services, use our Docker Compose setup that includes:

- **Multiple validator nodes**: Two independent Nibiru Chain nodes for testing multi-node scenarios
- **Pricefeeder services**: Automated price oracle data feeds
- **IBC relayer**: Cross-chain communication testing with Hermes relayer
- **Heart Monitor**: Blockchain indexing and GraphQL API for monitoring

See [contrib/docker-compose/README.md](./contrib/docker-compose/README.md) for detailed setup instructions and usage examples.


## How to Install Nibiru

Assuming you already have Go installed and common tools like `gcc` and `jq`, the
only commands you need to run are: 
```bash
just install  # to build the node software to make a Nibiru binary
just localnet # to run a local instance of Nibiru as a live network
```

For installation instructions from scratch, please see [INSTALL.md](./INSTALL.md).

Usage instructions for the `nibid` CLI are available at [nibiru.fi/docs/dev/cli](https://nibiru.fi/docs/dev/cli) and the [Nibiru Module Reference](https://nibiru.fi/docs/concepts/arch/#modules-%E2%80%94-nibiru).

<details>
<summary>[Recommended minimum specs to run a full node]</summary>

- 2CPU, 4GB RAM, 100GB SSD
- Unix system: MacOS or Ubuntu 18+

</details>

## License

Unless a file notes otherwise, it will fall under the [BSD-2-Clause License](./LICENSE.md).  

[license-badge]: https://img.shields.io/badge/License-BSD--2--Clause-blue
[badge-go-linter]: https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml/badge.svg?query=branch%3Amain
[workflow-go-linter]: https://github.com/NibiruChain/nibiru/actions/workflows/golangci-lint.yml?query=branch%3Amain
[badge-go-releaser]: https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml/badge.svg?query=branch%3Amain
[workflow-go-releaser]: https://github.com/NibiruChain/nibiru/actions/workflows/goreleaser.yml?query=branch%3Amain
[social-twitter]: https://x.com/NibiruChain
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
