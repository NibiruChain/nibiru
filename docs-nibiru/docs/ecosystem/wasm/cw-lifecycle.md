---
order: 2
---

# Lifecycle of a Contract

## What is a smart contract?

A CosmWasm **contract** is simply wasm bytecode uploaded to the chain. The contract
itself is not its "state". Think of the contract as a class. Contracts are
immutable because their code/logic is fixed.

Upon contract instantiation, there exists an **instance of the contract**. The
instance contains a reference to the contract and some state in a uniquely
prefixed data store, which is created upon contract instantiation. Concisely,
we might say, `contract (instance) = code + state`. In other words, an instance
of a contract is uniquely defined by a reference to wasm bytecode and a
reference to a prefixed key-value store. Instances are mutable because of state
changes.

## Lifecycle of a Contract:

1. Create contract code.
2. Instantiate a contract instance.
3. Invoke the instance.

> NOTE: Often, when we say "the contract", what is really intended is "a
particular instance of the contract". For brevity's sake, if we say something
like "contract state", "contract call", or "contract account", keep in mind
that it really means "contract *instance* state", "contract *instance* call",
or "contract *instance* account."

## Contract State

Contract state, or instance state, is mutable only by the one instance of the
contract but may be immutably read by any instance of the contract. This state is
a key-value store with one or more keys. State is born upon contract
instantiation.

- State is not "stored in" the contract, although it may feel this way during
  smart contract development. State comes from the Golang runtime (the chain).
- State is mutable only from the corresponding instance.

## EVM Tooling Analogs

| Ethereum | Wasm | Description |
| --- | --- | --- |
| Solidity, Vyper | CosmWasm-std + Rust | Smart contract languages |
| Open Zepplin contracts | cw-plus | Smart contract libraries with reusable components. |
| EVM | CosmWasm VM | Virtual machines that execute smart contract bytecode. |
| Web3.js | CosmJS | Client-side JavaScript libraries for interacting with blockchains. |

The [`cosmwasm-std` crate](https://docs.rs/cosmwasm-std/latest/cosmwasm_std/)
is the standard library for building contracts in CosmWasm. It is compiled as
part of the contract to Wasm. When creating a dependency to cosmwasm-std, the
required Wasm imports and exports are created implicitely via C interfaces.


<!-- TODO Q: Responsibilities of the `wasmvm` libraries -->
<!--  - [ ] Q: What does the Golang portion of the implementation do versus the Rust one?

To make the Wasm VM (CosmWasm) documentation more comprehensive, you should
address a variety of perspectives from the developer's to the end-user's needs.
Here are some sections/questions you might consider:

1. **Introduction to CosmWasm**
   - What is CosmWasm?
   - [ ] How does it differentiate from other Wasm systems?

2. **Getting Started**
   - How can someone set up and run CosmWasm?
   - Are there any prerequisites or dependencies?

3. **Detailed Architecture**
   - How does CosmWasm internally work?
   - What are the key components and their functions?

4. **Integration with Other Systems**
   - How does CosmWasm integrate with other blockchain systems or platforms?
   - Any specific configurations or adjustments needed?

5. **Security Considerations**
   - How does CosmWasm ensure the safe execution of smart contracts?
   - What measures are in place against potential threats?

6. **Development & Deployment**
   - How can developers write smart contracts for CosmWasm?
   - Are there any specific tools or SDKs available?
   - What's the deployment process for these contracts?

7. **Performance and Scalability**
   - How does CosmWasm handle high transaction volumes?
   - Are there any benchmarks available?

8. **Examples & Use Cases**
   - Are there notable projects or products built using CosmWasm?
   - How have they leveraged the features of CosmWasm?

9. **Troubleshooting & FAQ**
   - Common issues users/developers might encounter and their solutions.

10. **Community & Support**
   - How can someone get involved in the CosmWasm community?
   - Are there forums, chat groups, or other community resources?

11. **Roadmap & Future Plans**
   - What's next for CosmWasm?
   - Any upcoming features or improvements?

12. **Appendices**
   - Glossary of terms related to CosmWasm and Wasm in general.
   - Any other references or deep dives into specific topics.

-->


