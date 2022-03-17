
<p align="center">
  <img src="./matrix-logo.svg" width="300">
</p>
<h3 align="center">Decentralized Reserve Currency</h3>

<div align="center">

[![version](https://img.shields.io/github/tag/matrix-labs/matrix.svg)](https://github.com/matrixdao/matrix/releases/latest)
[![CircleCI](https://circleci.com/gh/MatrixDao/matrix/tree/master.svg?style=shield)](https://circleci.com/gh/MatrixDao/matrix/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/matrixdao/matrix)](https://goreportcard.com/report/github.com/matrixdao/matrix)
[![API Reference](https://godoc.org/github.com/MatrixDao/matrix?status.svg)](https://godoc.org/github.com/MatrixDao/matrix)
[![GitHub](https://img.shields.io/github/license/matrix-labs/matrix.svg)](https://github.com/MatrixDao/matrix/blob/master/LICENSE.md)
[![Twitter Follow](https://img.shields.io/twitter/follow/matrix_platform.svg?label=Follow&style=social)](https://twitter.com/matrix_platform)
[![Discord Chat](https://img.shields.io/discord/704389840614981673.svg)](https://discord.com/invite/pgArXgAxDD)

</div>

<div align="center">

### [Telegram](https://t.me/matrixlabs) | [Medium](https://medium.com/matrix-labs) | [Discord](https://discord.gg/pgArXgAxDD)

</div>

Matrix presents decentralized, over-collateralized and capital-efficient reserve protocol using [Cosmos SDK](https://github.com/cosmos/cosmos-sdk). Matrix enables liquid, capital efficient convertibility between stable assets and collateral using front-running resistant oracle to achieve the swapping.

TODO: Update installation instructions  

## Installation

Recommended minimum specs
- 2CPU
- 4GB RAM
- 100GB SSD
- Ubuntu 20.04 LTS

### Install Go (1.16+)

```
snap install --classic go
```

### Install Git

```
sudo apt install -y git gcc make
```

### Set the environment
```
sudo nano $HOME/.profile
```
Add the following 2 lines at the end of the file.
```
GOPATH=$HOME/go
PATH=$GOPATH/bin:$PATH
```
Save the file and exit the editor.
```
source $HOME/.profile
```

### Clone the Matrix Repository

```
git clone https://github.com/MatrixDAO/matrix
cd matrix
git checkout v0.0.1
make install
```

### Other recommended steps

- Increase number of open files limit
- Set your firewall rules

### Upgrade

The scheduled mainnet upgrade to `matrix-2` is planned for 

```
cd matrix
git fetch tags
git checkout v0.0.1
make install
```

## Testnet

One the Matrix binary has been installed, for further information on joining the testnet, head over to the [testnet repo](https://github.com/MatrixDao/Networks/tree/main/Testnet).

## Mainnet

One the Matrix binary has been installed, for further information on joining mainnet, head over to the [mainnet repo](https://github.com/MatrixDao/Networks/tree/main/Mainnet).

## Docs

Matrix protocol and client documentation can be found in the [Matrix docs](https://docs.matrix.io).

If you have technical questions or concerns, ask a developer or community member in the [Matrix discord](https://discord.com/invite/pgArXgAxDD).

## License

Copyright © Matrix Labs, Inc. All rights reserved.

Licensed under the [Apache v2 License](LICENSE.md).
