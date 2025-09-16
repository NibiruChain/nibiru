---
metaTitle: "Wasm | Quickstart"
order: 2
---

# Quickstart (Wasm)

Set up your development environment to interact with CosmWasm smart contracts on the Nibiru Chain. Learn to install Rust, set up the Wasm compiler, optimize contracts, and install the Nibiru CLI. Connect to the Nibiru testnet for deployment and testing.{synopsis}

## Setup Environement

To interact with a CosmWasm smart contract, you'll need to have Rust installed on your computer.
If you haven't installed it yet, you can find the installation instructions on the [Rust website](https://www.rust-lang.org/tools/install)
or use the recommended installation method below.

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

Assuming your rust installation is complete, you will require the Wasm rust compiler in order
to build Wasm binaries from your smart contracts.

```bash
rustup target add wasm32-unknown-unknown
```

Finally, gas fees for deploying smart contracts onto Nibiru are dependant on the size of your binaries.
It is highly recommended to optimize and minimize your contracts by using [**CosmWasm Rust Optimizer**](https://github.com/CosmWasm/optimizer).
This allows you to implement complex smart contracts without exceeding a size limit.
To install, you first need [docker](https://docs.docker.com/get-docker/).

Then you should be able to optimize your contracts using:

```bash
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.15.0
```

## Installing Nibiru CLI

For smart contract management, you have the option to use [`niibjs`](../tools/kickstart.md)
or the [`nibid`](../cli/README.md). For this guide, we will be utilizing `nibid`.

For installation, you can use the `curl` command below. For other installations methode please
refer to [Nibiru Binary Installation Guide](../cli/nibid-binary.md).

```bash
curl -s https://get.nibiru.fi/! | bash
```

## Setup Nibiru Testnet

Nibiru maintains public testnets to function as beta-testing environments as well as
testing playground for developers to test their dApp and Smart Contracts.

Documentation on connecting Nibiru's networks can be found [here](../networks).
For the purpose of this guide, to connect to Nibur's most stable network, testnet-1,
run the following:

```bash
RPC_URL="https://rpc.testnet-1.nibiru.fi:443"
nibid config node $RPC_URL
nibid config chain-id nibiru-testnet-1
nibid config broadcast-mode sync
nibid config # Prints your new config to verify correctness
```

## Related Pages

- [Nibiru Networks](../networks)
- [Nibiru Binary Installation Guide](../cli/nibid-binary.md)
