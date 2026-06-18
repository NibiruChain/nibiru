---
order: 3
---

# EVM Precompiles

## Introduction

Ethereum Virtual Machine (EVM) precompiles are special contract addresses that execute predefined native operations more efficiently than equivalent Solidity implementations. On Nibiru, specific precompiles enable interactions between EVM and Cosmos functionalities, such as transferring ERC-20 tokens to Cosmos bank coins and executing Wasm contracts.

This guide covers two essential precompiles on Nibiru:

- **FunToken Precompile**: Enables interactions between ERC-20 tokens and the Cosmos Bank module.
- **Wasm Precompile**: Facilitates executing Wasm contracts from the EVM.

## FunToken Precompile

### Address

```solidity
address constant FUNTOKEN_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000800;
IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);
```

### Interface

```solidity
interface IFunToken {
    function sendToBank(address erc20, uint256 amount, string calldata to) external returns (uint256 sentAmount);
    function balance(address who, address funtoken) external view returns (uint256 erc20Balance, uint256 bankBalance, FunToken memory token, NibiruAccount memory whoAddrs);
    function bankBalance(address who, string calldata bankDenom) external view returns (uint256 bankBalance, NibiruAccount memory whoAddrs);
    function whoAmI(string calldata who) external view returns (NibiruAccount memory whoAddrs);
    function sendToEvm(string calldata bankDenom, uint256 amount, string calldata to) external returns (uint256 sentAmount);
    function bankMsgSend(string calldata to, string calldata bankDenom, uint256 amount) external returns (bool success);
}
```

### Key Functions

1. **sendToBank**: Converts ERC-20 tokens into Cosmos bank coins.
2. **balance**: Retrieves the balances of both ERC-20 and Cosmos bank coins for a given account.
3. **bankBalance**: Fetches the Cosmos bank balance for a given address.
4. **whoAmI**: Resolves an address in both Bech32 (Nibiru) and EVM formats.
5. **sendToEvm**: Converts Cosmos bank coins into ERC-20 tokens.
6. **bankMsgSend**: Sends Cosmos bank coins to another address.

### Example Contracts

#### Example 1: Sending ERC-20 Tokens to Cosmos Bank

```solidity
pragma solidity ^0.8.0;

contract FunTokenExample {
    IFunToken constant funToken = IFunToken(0x0000000000000000000000000000000000000800);
    
    function convertToBank(address erc20, uint256 amount, string calldata cosmosAddress) external {
        funToken.sendToBank(erc20, amount, cosmosAddress);
    }
}
```

#### Example 2: Retrieving Balances

```solidity
pragma solidity ^0.8.0;

contract FunTokenBalanceChecker {
    IFunToken constant funToken = IFunToken(0x0000000000000000000000000000000000000800);
    
    function getBalances(address user, address token) external view returns (uint256, uint256) {
        (uint256 erc20Bal, uint256 bankBal,,) = funToken.balance(user, token);
        return (erc20Bal, bankBal);
    }
}
```

## Wasm Precompile

### Address

```solidity
address constant WASM_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000802;
IWasm constant WASM_PRECOMPILE = IWasm(WASM_PRECOMPILE_ADDRESS);
```

### Interface

```solidity
interface IWasm {
    function execute(string memory contractAddr, bytes memory msgArgs, INibiruEvm.BankCoin[] memory funds) external payable returns (bytes memory response);
    function executeMulti(WasmExecuteMsg[] memory executeMsgs) external payable returns (bytes[] memory responses);
    function query(string memory contractAddr, bytes memory req) external view returns (bytes memory response);
    function queryRaw(string memory contractAddr, bytes memory key) external view returns (bytes memory response);
    function instantiate(string memory admin, uint64 codeID, bytes memory msgArgs, string memory label, INibiruEvm.BankCoin[] memory funds) external payable returns (string memory contractAddr, bytes memory data);
    function migrate(string memory contractAddr, uint64 newCodeID, bytes memory msgArgs) external payable returns (bytes memory response);
}
```

### Key Functions

1. **execute**: Executes a Wasm contract's `ExecuteMsg` from the EVM.
2. **executeMulti**: Executes multiple Wasm contract calls in a single transaction.
3. **query**: Queries a Wasm contract using the `WasmQuery::Smart` variant.
4. **queryRaw**: Queries raw key-value storage of a Wasm contract.
5. **instantiate**: Instantiates a new Wasm smart contract.
6. **migrate**: Upgrades a Wasm smart contract to a new code version.

### Example Contracts

#### Example 1: Executing a Wasm Contract

```solidity
pragma solidity ^0.8.0;

contract WasmExecutor {
    IWasm constant wasm = IWasm(0x0000000000000000000000000000000000000802);
    
    function executeWasm(string memory contractAddr, bytes memory msgArgs, INibiruEvm.BankCoin[] memory funds) external payable {
        wasm.execute(contractAddr, msgArgs, funds);
    }
}
```

#### Example 2: Querying a Wasm Contract

```solidity
pragma solidity ^0.8.0;

contract WasmQueryContract {
    IWasm constant wasm = IWasm(0x0000000000000000000000000000000000000802);
    
    function queryWasm(string memory contractAddr, bytes memory queryMsg) external view returns (bytes memory) {
        return wasm.query(contractAddr, queryMsg);
    }
}
```

## Nibiru Codebase References

- [IFunToken.sol](https://github.com/NibiruChain/nibiru/blob/main/x/evm/embeds/contracts/IFunToken.sol)
- [Wasm.sol](https://github.com/NibiruChain/nibiru/blob/main/x/evm/embeds/contracts/Wasm.sol)
