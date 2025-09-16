---
order: 1
---

# Wasm Smart Contracts

Smart contracts for the Wasm VM on Nibiru are written in Rust and run in the
WebAssembly (Wasm) runtime. Wasm smart contracts are impervious to
re-entrancy attacks, the most common smart contract vulnerabilies in Ethereum,
are designed to be portable across IBC chains, and offer the memory safety and
performance benefits of Rust. {synopsis}

## Why Wasm Stands Out

CosmWasm offers a compelling set of features and advantages for smart contract
developers and users of Nibiru. Here's a closer look at some of these benefits
and why we love CosmWasm:

- **Security**: Defence against re-entrancy, denial of service attacks related
  to excessive gas consumption, and compile-time checks for overflow/underflow
  issues help make CosmWasm the ideal framework for developing secure,
  production-ready smart contracts.

- **Memory Safety**: CosmWasm contracts run in a sandboxed environment, ensuring that they can't
  harm the host system or access unintended data. The use of Rust, a memory-safe
  language, further enhances the security of contracts.

- **Performance**: Leveraging the efficiency of WebAssembly, CosmWasm can
  achieve near-native execution speeds for smart contracts, resulting in faster
  transaction processing. Here, "near-native execution" contracts  the
  performance of Wasm with the performance of code that runs directly on the
  hardware without any intermediate layers (native code). This is possible
  because Wasm code is low-level bytecoe that gets compiled nad optimized for
  the host machine before execution.

<!-- TODO write about testing  **Enhanced Developer Experience**: CosmWasm offers tools and structures
that simplify contract development and testing, providing a smoother experience
for developers. -->

- **Modularity and Flexibility**: CosmWasm is designed to be modular, allowing chains to adopt
  it without mandating its use for all smart contracts on the chain. CosmWasm
  is not just a layer on top of the Cosmos-SDK; it's deeply integrated,
  allowing for advanced features and tight interaction with the underlying
  blockchain infrastructure.

- **Interoperability**: CosmWasm contracts can execute cross-chain logic with the
  Inter-Blockchain Communication (IBC) protocol, enabling contracts to interoperate
  with other IBC-enabled chains.

- **Upgradable Contracts**: CosmWasm supports contract migration, allowing
  developers to upgrade their contracts post-deployment, a feature which many
  traditional smart contract platforms lack.

## Is it possible to interact with Wasm contracts on Nibiru using Ethereum wallets like MetaMask?

Yes, you can use Ethereum wallets like MetaMask to deploy and interact with both EVM and Wasm smart contracts on Nibiru. [Nibiru's Wasm precompile](../../evm/precompiles/index.md) allows seamless integration between the two virtual machines.

## Avoiding Reentrancy Attacks

CosmWasm smart contracts avoid all reentrancy attacks by design. This
point deserves an article by itself, but in short, [a large class of exploits in
Ethereum is based on this trick](https://consensys.github.io/smart-contract-best-practices/attacks/reentrancy/)
.

The idea is that in the middle of the execution of a function on Contract A,
it calls a second contract (explicitly or implicitly via send). This transfers
control to contract B, which can now execute code, and call into Contract A
again. Now there are two copies of Contract A running, and unless you are very,
very careful about managing the state before executing any remote contract or
make very strict gas limits in sub-calls, this can trigger undefined behavior
in Contract A and a clever hacker can reentrancy this as a basis for exploits,
such as the DAO hack.

Cosmwasm avoids this completely by preventing any contract from calling another
one directly. Clearly, we want to allow composition, but inline function calls
to malicious code create a security nightmare. The approach taken with CosmWasm
is to allow any contract to *return* a list of messages *to be executed in the
same transaction*.

This means that a contract can request a send to happen after it is finished
(eg. release escrow), or call into another contract. If the future messages
fail, then the entire transaction reverts, including updates to the contract's
state. This allows for atomic composition and quite a few security guarantees,
with the only real downside that you cannot view the results of executing
another contract, rather you can just do "revert on error".

## Lessons Learned from Ethereum

Ethereum is the grandfather of all blockchain smart contract platforms and has
far more usage and real-world experience than any other platform. We cannot
discount this knowledge but instead learn from their successes and failures to
produce a more robust smart contract platform.

They have compiled a list of [all known Ethereum attack
vectors](https://github.com/sigp/solidity-security-blog) along with mitigation
strategies. We shall compare Cosmwasm against this list to see how much of this
applies here. Many of these attack vectors are closed by design. A number
remain and a section is planned on avoiding the remaining such issues.

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


