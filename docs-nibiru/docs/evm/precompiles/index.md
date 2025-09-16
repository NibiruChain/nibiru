---
order: 0
metaTitle: "Precompiled Contracts (Precompiles)"
footer:
  newsletter: false
---

# Precompiles

Precompiled contracts, also called precompiles for short, extend the Nibiru EVM
to have functionality beyond what's present in the Ethereum Virtual Machine by
default. For example, Nibiru EVM contracts are interoperable with the Wasm VM and
can interact with and modify state outside the EVM. {synopsis}

::: tip
This page covers the basics on precompiles overall. You can skip ahead to the
following sections if you're already familiar with the fundamentals.
| Related Page | Synopsis |
| --- | --- |
| [Nibiru-Specific Precompiled Contracts](./nibiru.md) | Documentation on custom Nibiru precompiles |
| [Precompiles Common Between Ethereum and Nibiru](./ethereum.md) | Precompiled contracts that Nibiru inherits from Ethereum |
:::


## Intro to EVM Precompiled Contracts

Precompiled contracts are special contracts that exist at predefined addresses
and implement functions that are computationally expensive or require going
outside the standard set of operations in the EVM. 

These contracts are "precompiled" in the sense that their functionality is built
directly into the Ethereum Virtual Machine (EVM), allowing for more efficient
execution and the full flexibility of regular Golang logic.

## Common Characteristics of Precompiled Contracts

1. **EVM Interface**: Precompiles implement ABI methods and feel like normal contracts when you interact with them. 

2. **Fixed Addresses**: Each precompile is assigned a fixed address, allowing contracts to call them directly without the need for address resolution.

3. **Gas Cost Calculation**: Precompiles implement a `RequiredGas` function that calculates the gas cost based on the contract call `input`. This ensures that complex operations are appropriately priced.

4. **Deterministic Execution**: Like all EVM operations, precompiles must execute deterministically, producing the same output for the same input across all nodes.

Each precompile implements this interface from Geth.

```go
// PrecompiledContract is the basic interface for native Go contracts. The
// implementation requires a deterministic gas count based on the input size of
// the Run method of the contract.
type PrecompiledContract interface {
	ContractRef
	// RequiredPrice calculates the contract gas used
	RequiredGas(input []byte) uint64
	// Run runs the precompiled contract
	Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error)
}
```



## The Need for Custom Precompiles

Ethereum’s standard precompiles cover basic operations and useful cryptographic
functions, but more specialized functionalities are needed to cater to the use
cases of the Nibiru ecosystem. Custom precompiles allow for optimized, tailored
operations at the protocol level.

[Nibiru-specific precompiled contracts](./nibiru.md) address several key needs:

1. **Performance Optimization**: Implementing complex operations directly in the
   VM can significantly reduce gas costs and improve overall network efficiency.
2. **Purpose-built Custom Functionality**: Nibiru has a unique architecture and
   functionality that caters specifically to the Nibiru ecosystem. The EVM is not
the only area of Nibiru that holds state. In order to express this external,
non-EVM state or modify it from the EVM, there needs to be smart contracts that
makes it possible to do so. Precompiles are those contracts.
3. **Enhanced Interoperability**: Custom precompiles can facilitate seamless interaction
   between different blockchain environments or virtual machines. The toolkit and
   application surface available to developers 
4. **Modular Account Operations**: Account types that were not created on the
   EVMare important to the overall blockchain state. Examples include standard
`BaseAccounts` created from interchain wallets and Wasm smart contract accounts.
5. **Asset Composability**: The ability to easily convert between ERC20 tokens
   and a canonical Bank Coin representation opens up possibilities for cross-VM
applications and opens up Ethereum functionality to the broader interchain/IBC
landscape. For instance, Nibiru acts as a hub for officially supported USDC from Circle similar like Noble and can open LayerZero assets in the same manner.


## The Significance of Nibiru's Approach

Nibiru's implementation of custom precompiles represents a significant leap forward in blockchain interoperability and functionality. By bridging the EVM and Cosmos ecosystems, Nibiru offers several key advantages:

1. **Enhanced Interoperability**: Developers can create applications that seamlessly utilize features from both EVM and Cosmos environments.
2. **Expanded Functionality**: EVM contracts can access Cosmos-SDK features, and vice versa, greatly expanding the toolkit available to developers.
3. **Improved Performance**: By implementing complex cross-environment operations at the protocol level, Nibiru achieves significant performance improvements and gas efficiency.

<!-- ## Implementing and Using Nibiru Precompiles 

TODO: Document how to import and use the precompiles in TS
TODO: Document how to import and use the precompiles in Solidity
https://github.com/NibiruChain/ts-sdk/issues/376

For developers looking to leverage Nibiru's custom precompiles, the process is straightforward and similar to interacting with standard Ethereum precompiles. Here's a simple example of using the FunToken precompile to send ERC20 tokens as bank coins:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IFunToken.sol";

contract FunTokenExample {
    IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(0x0000000000000000000000000000000000000800);

    function sendToNibiruAddress(address erc20, uint256 amount, string memory to) external {
        FUNTOKEN_PRECOMPILE.bankSend(erc20, amount, to);
    }
}
```

Similarly, here's how you might use the Wasm precompile to execute a function on a Wasm contract:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IWasm.sol";

contract WasmExample {
    IWasm constant WASM_PRECOMPILE = IWasm(0x0000000000000000000000000000000000000802);

    function executeWasmContract(string memory contractAddr, bytes memory msgArgs) external {
        WASM_PRECOMPILE.execute(contractAddr, msgArgs, new IWasm.BankCoin[](0));
    }
}
```

-->


When working with Nibiru’s precompiles, developers should keep the following in mind:

1. **Address Formats**: Be aware of the differences between the hexadecimal Ethereum address and "nibi"-prefixed Bech32 address formats.

## Future Implications and Possibilities

Nibiru's innovative use of custom precompiles opens up exciting possibilities for the future of blockchain technology:

1. **Further Custom Precompiles**: There's potential for additional precompiles that could further enhance interoperability, such as IBC (Inter-Blockchain Communication) precompiles.
2. **Multi-VM Architectures**: Nibiru's approach could serve as a model for future blockchain platforms looking to integrate multiple virtual machine environments.

## Read Next

- [Nibiru-Specific Precompiled Contracts](./nibiru.md)
- [Precompiles Common Between Ethereum and Nibiru](./ethereum.md)
- [Execution Engine](../../arch/execution.md)
