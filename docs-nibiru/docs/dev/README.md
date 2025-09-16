# Developer Hub

Everything you need to build on Nibiru. Your go-to hub to develop smart contracts applications for the decentralized web. {synopsis}

<img src="../img/decor-1-trees.png" style="border-radius: 16px;">

## Nibiru Networks & RPC Endpoints  

Easily configure your wallet, node, or dApp with Nibiru's network settings. Below are the RPC endpoints, network identifiers, and blockchain explorers for both the **mainnet** and **testnet**. For a full setup guide, including adding Nibiru as a custom network in your wallet, visit: [Nibiru Networks and RPCs](./networks/README.md).  

| EVM Info | Nibiru **Mainnet**  | Nibiru **Testnet** |
| --- | --- | --- |
| **EVM RPC** | [evm-rpc.nibiru.fi](https://evm-rpc.nibiru.fi) | [evm-rpc.testnet-2.nibiru.fi](https://evm-rpc.testnet-2.nibiru.fi) |
| **EIP-155 Chain ID** | `6900` (`0x1AF4` in Hex) | `6911` (`0x1AFF` in Hex) |
| **CSDK Chain-ID** | `cataclysm-1` | `nibiru-testnet-2` |
| **EVM Explorer** | [nibiscan.io/](https://nibiscan.io/) | [testnet.nibiscan.io/](https://testnet.nibiscan.io/) |

## Core Tools and Language Clients

Essential SDKs and tools for building applications on Nibiru.

| Tool / Library | Description |
| --- | --- |
| [@nibiruchain/nibijs](./tools/nibijs/README.md) | A TypeScript SDK for building web applications in frameworks like Vue and React. It also supports interacting with wallet extensions like Keplr and MetaMask. Published as an npm package. |
| [@nibiruchain/evm-core](./evm/npm-evm-core.md) | Core Nibiru EVM (Ethereum Virtual Machine) TypeScript library with functions to call Nibiru-specific precompiled contracts. |
| [@nibiruchain/solidity](./evm/npm-solidity.md) | Nibiru EVM solidity contracts and ABIs for Nibiru-specific precompiles and core protocol functionality. |
| [Nibiru Rust SDK](./cw/rust-sdk.md) | Nibiru Rust standard library that includes the `nibiru-std` crate, offering protobuf types and traits for [**Wasm (WebAssembly)** smart contracts](./cw/README.md) on Nibiru. Available on crates.io. |
| [Nibiru Command-Line Interface (CLI)](./cli/README.md) | The **`nibid` binary is needed for running nodes** and sending IBC transfers without using a wallet extension. |

## Oracle Solutions

1. [Nibiru Oracle (EVM) - Usage Guide](./evm/oracle.md): Technical guide for using the Nibiru Oracle precompile in EVM smart contracts, including ChainLink-like price feeds.
2. [Integrating with Oracles on Nibiru](./tools/oracle/index.md): Overview of available oracle solutions on Nibiru, including the native oracle and Band Protocol integration.

## Block Explorers

[Nibiru Block Explorers](../community/explorers.md): Block explorer for Nibiru

## Other Tools

- [Nibiru Golang SDK](./tools/go-sdk.md): Useful for developing agents and
applications while directly importing types directly from the Nibiru source code.
- [Nibiru Python SDK](./tools/py-sdk.md): Similar to the TypeScript SDK, except
written in Python.

- Testnet Faucet
  - [Testnet Faucet (Repo)][repo-faucet]: Send tokens to your wallet on testnet
  - [Usage Example GitHub Gist](https://gist.github.com/Unique-Divine/f2692c42a758afb98db55be3c4304f40#file-04_faucet-sh)

    <!-- ```bash
    FAUCET_URL="https://faucet.itn-1.nibiru.fi/"
    ADDR="..." # ← with your address
    curl -X POST -d '{"address": "'"$ADDR"'", "coins": ["11000000unibi","100000000unusd","100000000uusdt"]}' $FAUCET_URL
    ``` -->

[repo-faucet]: https://github.com/NibiruChain/faucet

## Undersanding the Nibiru Blockchain

See the [Nibiru Architecure](../arch/README.md) for comprehensive documentation.

Nibiru's Cosmos-SDK modules define the core logic for Nibi-Perps, Nibi-Swap, and the decentralized oracle network. Nibiru's modules are defined in the "x/" subfolder of the protocol's Golang code (e.g. the "evm" module is defined in the "x/evm" folder).

## Nibiru Discord Server

If you would like to connect with the developer community and ask questions related to software development on Nibiru, join the [Nibiru Discord server][discord-url]. Once you've joined the server:

[discord-url]: https://discord.gg/HFvbn7Wtud
