---
order: false # TODO
#order: 7
---

# CosmWasm Bindings

Bindings refer to a layer of code that allows two different programming
languages or systems to communicate with each other.

Specifically, in the context of CosmWasm smart contracts and the Cosmos SDK,
bindings allow the Go-based Cosmos SDK to interact with and execute smart
contracts written in Rust and compiled to WebAssembly (Wasm).

1. The Cosmos SDK is a Golang framework for building blockchains. Defines
   various modules such as `x/bank` and `x/staking` that provide functionality
   for building ddecentralized applications.

2. CosmWasm is a framework for writing secure smart contracts in the Rust
   programming language, which are then compiled to WebAssembly.

The bindings between the Cosmos SDK and CosmWasm smart contracts serve to
translate between the two. This includes:

1. Calling into the Wasm virtual machine from Go

2. Translating between Go types and Wasm types

3. Handling the translation between messages emitted by a Wasm contract and
   messages understood by the Cosmos SDK

Bindings serve as the bridge between these different systems, allowing them to
interact with and understand each other.

