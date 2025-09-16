---
order: 2
footer:
  newsletter: false
---

# Nibiru-Specific Precompiled Contracts

Precompiled contracts extend the Nibiru EVM to include behavior outside of what's
present on the Ethereum Virtual Machine by default. For example, Nibiru's
Execution Engine allows Nibiru EVM contracts to interoperate with the Wasm VM
using the Wasm precompile. And other Nibiru-specific precompiled contracts can
interact with and modify state outside the EVM. {synopsis}

## Reference List: All Custom Precompiles

| Precompile | Description | Address |
| --- | --- | --- |
| Wasm.sol | Enables the invocation of Wasm VM contracts  | 0x00...802 |
| FunToken.sol | Makes it possible to send ERC20 tokens as bank coins to a Nibiru bech32 address using the "FunToken" mapping between the ERC20 and bank coin. | 0x00...800 |
| [Oracle.sol](./oracle.md) | Implements the exchange rate query from Nibiru's Oracle Module. This makes low-latency price data accessible from smart contracts. | 0x00...801 |


## Wasm Precompile

The Wasm Precompile (Wasm.sol) of Nibiru allows EVM contracts to interact directly with Wasm-based smart contracts. This enables seamless communication between the EVM and Wasm portions of the execution engine.

The Wasm precompile offers several key methods:

| Wasm.sol Method | Description
| --- | --- |
| `execute` | Invokes Wasm contract functions from within the EVM environment.
| `query` | Enables smart queries of Wasm contracts.
| `instantiate` | Allows for the creation of new Wasm contract instances.
| `executeMulti` | Facilitates batch execution of multiple Wasm contract calls.
| `queryRaw` | Provides low-level querying of Wasm contract state using a key for the contract's key-value store.

These methods enable unprecedented interoperability between EVM and Wasm environments, allowing developers to leverage the strengths of both ecosystems within a single blockchain platform.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

address constant WASM_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000802;

IWasm constant WASM_PRECOMPILE = IWasm(WASM_PRECOMPILE_ADDRESS);

interface IWasm {
  struct BankCoin {
    string denom;
    uint256 amount;
  }

  /// @notice Invoke a contract's "ExecuteMsg", which corresponds to
  /// "wasm/types/MsgExecuteContract". This enables arbitrary smart contract
  /// execution using the Wasm VM from the EVM. 
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param msgArgs JSON encoded wasm execute invocation
  /// @param funds Optional funds to supply during the execute call. It's
  /// uncommon to use this field, so you'll pass an empty array most of the time.
  /// @dev The three non-struct arguments are more gas efficient than encoding a
  /// single argument as a WasmExecuteMsg.
  function execute(
    string memory contractAddr,
    bytes memory msgArgs,
    BankCoin[] memory funds
  ) payable external returns (bytes memory response);

  struct WasmExecuteMsg {
    string contractAddr;
    bytes msgArgs;
    BankCoin[] funds;
  }

  /// @notice Identical to "execute", except for multiple contract calls.
  function executeMulti(
    WasmExecuteMsg[] memory executeMsgs
  ) payable external returns (bytes[] memory responses);

  /// @notice Query the public API of another contract at a known address (with
  /// known ABI). 
  /// Implements smart query, the "WasmQuery::Smart" variant from "cosmwas_std".
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param req JSON encoded query request
  /// @return response Returns whatever type the contract returns (caller should
  /// know), wrapped in a JSON encoded contract result.
  function query(
    string memory contractAddr, 
    bytes memory req
  ) external view returns (bytes memory response);

  /// @notice Query the raw kv-store of the contract. 
  /// Implements raw query, the "WasmQuery::Raw" variant from "cosmwas_std".
  /// @param contractAddr nibi-prefixed Bech32 address of the wasm contract
  /// @param key contract state key. For example, a `cw_storage_plus::Item` of
  /// value `Item::new("state")` creates prefix store with key, "state".
  /// @return response JSON encoded, raw data stored at that key.
  function queryRaw(
    string memory contractAddr, 
    bytes memory key
  ) external view returns (bytes memory response);

  /// @notice InstantiateContract creates a new smart contract instance for the
  /// given code id.
  function instantiate(
    string memory admin,
    uint64 codeID,
    bytes memory msgArgs,
    string memory label,
    BankCoin[] memory funds
  ) payable external returns (string memory contractAddr, bytes memory data);

}
```

## FunToken Precompile

The FunToken precompile enables transfers between ERC20 tokens in the EVM environment and Bank Coins outside the EVM. This is enabled by the [creation of a `FunToken` (fungible token) mapping](../funtoken.md), as this provides a canonical tie between different token standards.

The key functionality of the FunToken precompile is encapsulated in its `bankSend` method:

```solidity
interface IFunToken {
  /// @dev bankSend sends ERC20 tokens as coins to a Nibiru base account
  /// @param erc20 the address of the ERC20 token contract
  /// @param amount the amount of tokens to send
  /// @param to the receiving nibi-prefixed bech32 address.
  function bankSend(address erc20, uint256 amount, string memory to) external;
}
```

This method allows users to send ERC20 tokens from the Nibiru EVM environment as Bank Coins to any other Nibiru account using its Bech32 address. This opens up a world of possibilities for multi-VM applications.
