# Artchitecture: CosmWasm

- wasmd: Implements the `x/wasm` module.

- cosmwasm-std: The standard library for building CosmWasm smart contracts. Code
  in this package is compiled into the smart contract.

- cosmwasm-vm (Rust Crate):
  [[Repo](https://github.com/CosmWasm/cosmwasm/tree/main/packages/vm)] : This is
  an abstraction layer around the wasmer VM to expose just what we need to run
  cosmwasm contracts in a high-level manner. This is intended both for efficient
  writing of unit tests, as well as a public API to run contracts in eg. wasmvm.
  As such it includes all glue code needed for typical actions, like fs caching.

- CosmWasm/wasmvm (GitHub): Go bindings to the running cosmwasm contracts with
  wasmer. This repo contains both Rust and Go code. The Rust code is compiled
  into a library (shared .dll/.dylib/.so or static .a) to be linked via cgo and
  wrapped with a pleasant Go API. The full build step involves compiling Rust ->
  C library, and linking that library to the Go code. For ergonomics of the user,
  we will include pre-compiled libraries to easily link with, and Go developers
  should just be able to import this directly.

## Layers of Abstraction in the system

- Wasmer VM / Runtime (Rust): A high-performance WebAssembly (Wasm) runtime
  written in Rust and the foundational layer upon which higher-level abstractions
  are built. Wasmer interprets and executes WebAssembly bytecode, which ensures
  the safe execution of smart contracts.
- CosmWasm VM (Rust): This is an abstraction layer around the wasmer VM to expose
  just what we need to run cosmwasm contracts. It ismeant to enable efficient
  writing of unit tests and expose a public API to run contracts.
- wasmvm (Rust + Golang): Abstraction layer or wrapper around around the CosmWasm
  VM (Rust). The `wasmvm` tool is what enables one to compile, initialize, and
  invoke CosmWasm smart contracts from Golang applications.
- `x/wasm`: A Cosmos-SDK module specifically meant to process transaction
  messages for CosmWasm smart contracts. From the app's perspective, this is the
  interface for the WasmVM, Wasmer, and CosmWasm smart contracts.
- `wasmd`: A specific example chain that uses the `x/wasm` module and the minimal
  subset of default modules from the Cosmos-SDK. The `wasmd` chain serves as a
  great example on how to integrate CosmWasm into the chain.

<!-- # wasmd -->
