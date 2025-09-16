---
order: 3
---

# Precompiles Common Between Ethereum and Nibiru

Not all EVM precompiles are custom to Nibiru. The Nibiru EVM includes essential
precompiled contracts from Ethereum based on its Berlin upgrade, which
includes functionality ranging from basic cryptographic operations  to more
complex functions like modular exponentiation and elliptic curve operations.
These precompiles offer gas-efficient deterministic execution available at fixed
addresses.

## Common Characteristics of Precompiled Contracts

1. **Fixed Addresses**: Each precompile is assigned a fixed address, allowing contracts to call them directly without the need for address resolution.

2. **Gas Cost Calculation**: Precompiles implement a `RequiredGas` function that calculates the gas cost based on the input. This ensures that complex operations are appropriately priced.

3. **EVM Interface**: All precompiles implement the `Run` method, which is called by the EVM when the precompile is invoked. This method takes the input data and returns the result along with any error.

4. **Deterministic Execution**: Like all EVM operations, precompiles must execute deterministically, producing the same output for the same input across all nodes.

## Precompiles Common Between Nibiru and Ethereum

The `PrecompiledContractsBerlin` set represents the state of Ethereum precompiles as of the Berlin hard fork. This set includes all precompiles from previous versions and introduces optimizations and new functionalities.

Here's a comprehensive list of the precompiles included in this set:

| Name | Address | Purpose |
|---------|------|---------|
| ECRECOVER | 0x01 | Elliptic curve public key recovery |
| SHA256 | 0x02 | SHA-256 hash function |
| RIPEMD160 | 0x03 | RIPEMD-160 hash function |
| IDENTITY | 0x04 | Data copy operation |
| MODEXP | 0x05 | Modular exponentiation |
| ECADD | 0x06 | Elliptic curve addition |
| ECMUL | 0x07 | Elliptic curve scalar multiplication |
| ECPAIRING | 0x08 | Elliptic curve pairing check |
| BLAKE2F | 0x09 | BLAKE2 compression function |

The Berlin upgrade, implemented in Ethereum's hard fork in April 2021, brought significant improvements to gas cost calculations for existing precompiles and introduced the BLAKE2F precompile. These enhancements aimed to optimize network performance and expand the cryptographic capabilities available to smart contracts.


## Introduction to Shared Precompiles

Precompiled contracts are a fundamental component of the Ethereum Virtual Machine (EVM), providing optimized implementations of frequently used cryptographic operations. These contracts reside at predefined addresses and offer significant gas savings compared to equivalent operations implemented in Solidity.

Nibiru, in its commitment to EVM compatibility, adopts the `vm.PrecompiledContractsBerlin` set, ensuring that developers can leverage the same optimized operations available in Ethereum's Berlin upgrade. This compatibility not only enhances performance but also facilitates the seamless migration of Ethereum dApps to the Nibiru ecosystem.

