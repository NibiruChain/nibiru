# nibiru-std

> Nibiru standard library for CosmWasm smart contracts.

```bash
⚡ Nibiru Standard Library Packages
├── 📦 nibiru-std              # Nibiru standard library for smart contracts
├── 📦 nibiru-ownable          # Utility for single-party ownership of CosmWasm smart contracts
└── 📦 nibiru-ownable-derive   # Macros for generating code used by the `nibiru-ownable` crate
```

`nibiru-std` enables smart contracts to send a multitude of Nibiru-specific transactions from Wasm contracts, or with independent clients. This library provides types and traits for `QueryRequest::Stargate` and `CosmosMsg::Stargate`, including constructors for Cosmos, IBC, and Nibiru protocol messages.

## Features

- **Stargate Integration**: Send custom Cosmos SDK messages through CosmWasm's Stargate interface
- **Nibiru Protocol Support**: Direct integration with Nibiru-specific modules and transactions
- **Query Capabilities**: Execute complex queries against the Nibiru blockchain state
- **Type Safety**: Strongly-typed Rust interfaces for all supported message types

## Documentation

For detailed API documentation, visit [docs.rs/nibiru-std](https://docs.rs/nibiru-std).

## Repository

This package is part of the [NibiruChain/nibiru-wasm](https://github.com/NibiruChain/nibiru-wasm) monorepo, which contains smart contract examples and additional tooling for Nibiru development.