# Nibiru Chain          <!-- omit in toc -->

<!--  
<p align="center">
 <img src="./nibiru-logo.svg" width="300"> 
</p>
<h1 align="center">Nibiru Protocol</h1>
-->

[![Nibiru Test workflow][go-unit-tests-badge]][go-unit-tests-workflow]
[![GitHub][license-badge]](https://github.com/NibiruChain/nibiru/blob/master/LICENSE.md)
[<img align="right" alt="Nibiru Telegram" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@3.13.0/icons/telegram.svg" />][nibiru-telegram]
[<img align="right" alt="Personal Website" width="22px" src="https://raw.githubusercontent.com/iconic/open-iconic/master/svg/globe.svg" />][nibiru-website]
[<img align="right" alt="Nibiru Discord" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@v3/icons/discord.svg" />][nibiru-discord]
[<img align="right" alt="Nibiru Medium Blog" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@3.13.0/icons/medium.svg" />][nibiru-medium]

**Nibiru Chain** is a proof-of-stake blockchain and member of a family of interconnected blockchains that comprise the Cosmos Ecosystem. Nibiru powers three main decentralized applications:

- **Nibiru Perpetuals Exchange**: On the perps exchange, users can take leveraged exposure and trade on a plethora of assets: completely on-chain, completely non-custodially, and with minimal gas fees.
- **Nibiru AMM**: An automated market maker for multichain assets. This application gives users access to swaps, pools, and bonded liquidity gauges.
- **Nibiru Stablecoin**: Nibiru employs a two-token economic model, where NIBI is the governance and utility token for the protocol and NUSD is a capital-efficient, partially collateralized algorithmic stablecoin created by the protocol.

Nibiru is built with the [Cosmos-SDK][cosmos-sdk-repo], accurately prices assets using a system of front-run resistant oracles, and communicates with other Cosmos layer-1 chains using the [Inter-Blockchain Communication (IBC)](https://github.com/cosmos/ibc) protocol.  

### ⚙️ — Documentation

Conceptual and technical documentation can be found in the [Nibiru docs](https://docs.nibiru.io). Each module also contains a detailed specification in its "spec" directory (e.g. [`x/stablecoin/spec`](https://github.com/NibiruChain/nibiru/tree/master/x/stablecoin/spec)).

### 💬 — Community

If you have questions or concerns, feel free to connect with a developer or community member in the [Nibiru discord][nibiru-discord]. We also have active communities on Twitter and Telegram.

<!-- Markdown versions of the social badges 
[![description][discord-badge]][nibiru-discord] 
[![description][twitter-badge]][nibiru-twitter] 
[![description][telegram-badge]][nibiru-telegram]
-->

<p style="text-align:right">
<a href="https://discord.com/invite/pgArXgAxDD"><img src="https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white" alt="Discord" height="22"/></a>
<a href="https://twitter.com/NibiruChain"><img src="https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white" alt="Tweet" height="22"/></a>
<a href="example.com"><img src="https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white" alt="Telegram" height="22"/></a>
</p>

----

## ⛓️ Installation: Developing on the chain locally

Installation instructions for the `nibid` binary can be found in [INSTALL.md](./INSTALL.md).

Recommended minimum specs:

- 2CPU, 4GB RAM, 100GB SSD
- Unix system: MacOS or Ubuntu 18+

## Developing on the chain locally

### Single Node

On a fresh clone of the repo, simply run `make localnet` and open another terminal.  

### Generate the protobufs

```bash
make proto-gen
```

# Linter

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

Copyright © Nibiru Labs, Inc. All rights reserved.

Licensed under the [Apache v2 License](LICENSE.md).

[nibiru-medium]: example.com
[nibiru-telegram]: example.com
[nibiru-website]: https://github.com/NibiruChain
[license-badge]: https://img.shields.io/badge/License-Apache_2.0-blue.svg
[cosmos-sdk-repo]: https://github.com/cosmos/cosmos-sdk
[go-unit-tests-badge]: https://github.com/NibiruChain/nibiru/actions/workflows/go.yml/badge.svg
[go-unit-tests-workflow]: https://github.com/NibiruChain/nibiru/actions/workflows/go.yml
[nibiru-twitter]: https://twitter.com/NibiruChain
[nibiru-discord]: https://discord.com/invite/pgArXgAxDD

[discord-badge]: https://img.shields.io/badge/Discord-7289DA?&logo=discord&logoColor=white
[twitter-badge]: https://img.shields.io/badge/Twitter-1DA1F2?&logo=twitter&logoColor=white
[telegram-badge]: https://img.shields.io/badge/Telegram-2CA5E0?&logo=telegram&logoColor=white

<!--
[![Twitter Follow](https://img.shields.io/twitter/follow/nibiru_platform.svg?label=Follow&style=social)][nibiru-twitter]

[![version](https://img.shields.io/github/tag/nibiru-labs/nibiru.svg)](https://github.com/NibiruChain/nibiru/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/NibiruChain/nibiru)](https://goreportcard.com/report/github.com/NibiruChain/nibiru) 

[![API Reference](https://godoc.org/github.com/NibiruChain/nibiru?status.svg)](https://godoc.org/github.com/NibiruChain/nibiru)

[![Discord Chat](https://img.shields.io/discord/704389840614981673.svg)][nibiru-discord]
-->