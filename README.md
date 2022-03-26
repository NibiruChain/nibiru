# Matrix Protocol

<!--  
<p align="center">
 <img src="./matrix-logo.svg" width="300"> 
</p>
<h1 align="center">Matrix Protocol</h1>
-->

[![Matrix Test workflow][go-unit-tests-badge]][go-unit-tests-workflow]
[![GitHub](https://img.shields.io/github/license/matrix-labs/matrix.svg)](https://github.com/MatrixDao/matrix/blob/master/LICENSE.md)
[![Discord Chat](https://img.shields.io/discord/704389840614981673.svg)][matrix-discord]
[<img align="right" alt="Matrix Telegram" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@3.13.0/icons/telegram.svg" />][Telegram]
[<img align="right" alt="Personal Website" width="22px" src="https://raw.githubusercontent.com/iconic/open-iconic/master/svg/globe.svg" />][matrix-website]
[<img align="right" alt="Matrix Discord" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@v3/icons/discord.svg" />][matrix-discord] 
[<img align="right" alt="Matrix Medium Blog" width="22px" src="https://cdn.jsdelivr.net/npm/simple-icons@3.13.0/icons/medium.svg" />][Medium]


Matrix is a zone in the Cosmos that houses a decentralized ecosystem  consisting of  3 main applications:  
① USDM: A partially collateralized, algorithmic stablecoin protocol  
② Derivatives: A platform for trading perpetual futures, enabling users to take long and short leveraged exposure in a capital efficient manner.   
③ DEX: An automated market maker for both standard swaps and "stable swaps" of multichain assets.  

Matrix is built with the [Cosmos SDK][cosmos-sdk-repo], Tendermint Consensus, and a system of front-run resistant oracles for accurate pricing on the stablecoin, dex, and derivatives applications. 

#### Documentation 

- Conceptual and technical documentation can be found in the [Matrix docs](https://docs.matrix.io).
- If you have questions or concerns, feel free to connect with a developer or community member in the [Matrix discord][matrix-discord].

[Medium]: example.com
[Telegram]: example.com
[matrix-website]: https://github.com/MatrixDao
[cosmos-sdk-repo]: https://github.com/cosmos/cosmos-sdk
[go-unit-tests-badge]: https://github.com/MatrixDao/matrix/actions/workflows/go.yml/badge.svg
[go-unit-tests-workflow]: https://github.com/MatrixDao/matrix/actions/workflows/go.yml
[matrix-twitter]: https://twitter.com/matrix_platform 
[matrix-discord]: https://discord.com/invite/pgArXgAxDD
  

<!--
[![Twitter Follow](https://img.shields.io/twitter/follow/matrix_platform.svg?label=Follow&style=social)][matrix-twitter]
[![version](https://img.shields.io/github/tag/matrix-labs/matrix.svg)](https://github.com/matrixdao/matrix/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/matrixdao/matrix)](https://goreportcard.com/report/github.com/matrixdao/matrix) 
[![API Reference](https://godoc.org/github.com/MatrixDao/matrix?status.svg)](https://godoc.org/github.com/MatrixDao/matrix)
-->

----

## Installation

Installation instructions can be found here: [INSTALL.md](./INSTALL.md).

Recommended minimum specs:
- 2CPU, 4GB RAM, 100GB SSD
- Unix system: MacOS or Ubuntu 18+

## Developing on the chain locally

On a fresh clone of the repo, simply run `make localnet` and open another terminal.  


## License

Copyright © Matrix Labs, Inc. All rights reserved.

Licensed under the [Apache v2 License](LICENSE.md).
