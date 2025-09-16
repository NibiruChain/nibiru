---
metaTitle: "Nibiru EVM | Quickstart"
order: 1
---

# Quickstart

Set up your development environment to interact with the Nibiru EVM on the local Nibiru Chain. This guide covers installing Rust & GO, setting and running a local Nibiru network for deployment and testing.{synopsis}

## 1. Install Rust & Just

If you haven't installed Rust yet, you can find the installation instructions on the [Rust website](https://www.rust-lang.org/tools/install) or use the recommended installation method below.

Step-by-Step Installation:

1. Run the following command to install Rust:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

2. Install just, a command runner for Rust projects:

```bash
cargo install just
```

## 2. Install Go

Nibiru requires Go for its build processes. Follow the steps below to install Go (v1.18) for Unix-based systems like macOS, Ubuntu, or WSL. Please install Go v1.18 using the instructions at [go.dev/doc/install](https://go.dev/doc/install).

Installation Instructions for Ubuntu:

1. Download and install Go:

```bash
wget https://golang.org/dl/go1.18.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
```

2. Set environment variables in your shell configuration or add to your `.zshrc` or `.bashrc`:

```bash
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export GO111MODULE=on
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
```

## 3. Run a Local Nibiru Chain

#### 3.1 Clone the Nibiru Repository

Start a local Nibiru network by cloning the NibiruChain/nibiru repository and checking out the latest stable version.

```bash
cd $HOME
git clone https://github.com/NibiruChain/nibiru
cd nibiru
git checkout v2.0.0-evm.2  # Use "main" for the latest development version
```

#### 3.2 Nibiru Commands

Use `just` to see availble commands, one of them being `just localnet`. This will create a localnet for you

```bash
cd nibiru
just localnet
```

With the Localnet running, you can open a new terminal and can full interact with the local chain via local rpc endpoint and pre-existing accounts from the genisis.

**Note**: If facing dependency or library issues, remove the `/temp` folder and rerun localnet.

```bash
JSON_RPC_ENDPOINT="http://127.0.0.1:8545"
MNEMONIC="guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
```

::: tip
Notice that that default port for the Ethereum JSON RPC is 8545. 
:::

#### 3.3 Validate setup [optional]

Ensure setup correctness with Nibiru's EVM end-to-end tests.

1. Navigate to the EVM e2e test directory and install dependencies:

```bash
cd nibiru/e2e/evm
npm install
cp .env_sample .env
```

2. Run the tests:

```bash
npm test
```

## 4. Local Explorer [Optional]

After spinning up the localnet, you can spin up a local explorer in order to better view local transactions and to test your smart contract. There are several optons available online, below is outlined one of those options with their easy setup.

[Etherparty](https://github.com/etherparty/explorer)

1. Clone the repo using git

```bash
git clone git@github.com:etherparty/explorer.git
```

2. Change your Node version to `v18` via `nvm`

```bash
cd explorer
nvm install 18
nvm use 18
nvm version
```

3. Install dependancies

```bash
npm install
```

4. Serve application to default `http://localhost:8000/`

```bash
npm start
```

> **_NOTE:_**  The default EVM RPC within the explorer is the same default local evm on nibiru `http://localhost:8545` and it is hardcoded under `app/app.js`.

## Related Pages

- [Nibiru EVM](../../evm/README.md)
- [Nibiru Networks](../networks)
- [Nibiru Source Code Installation Guide](../cli/nibid-binary.md#install-option-3--building-from-the-source-code)
