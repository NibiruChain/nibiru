# @nibiruchain/solidity

Nibiru EVM solidity contracts and ABIs for Nibiru-specific precompiles and core protocol functionality.

## Install

```bash
yarn add @nibiruchain/solidity
# OR npm install OR bun install
```

Solidity code is in "`@nibiruchain/solidity/contracts/*`", and 
ABI JSON files are in "`@nibiruchain/solidity/abi/*`".

## ABI Files 

Since contract ABIs are exported in JSON format, these ABIs can be used with any
Ethereum-compatible JS library, such as:
- Viem
- Ethers.js
- Web3.js
- Hardhat or Foundry

```ts
// Example import
import wasmAbi from "@nibiruchain/solidity/abi/IWasm.json"
```

## Usage with Typechain for Ethers v6

Often when building an application using Ethers.js, we want type-safe contract
interactions. The `typechain` and `@typechain/ethers-v6` packages are a useful
tool to generate TypeScript code an `ethers.Contract` directly from smart
contract ABIs.

To generate `ethers` TypeScript bindings for the ABIs exported from
the `@nibiruchain/solidity` package, first install the required dependencies:
```
npm install -D @nibiruchain/solidity @typechain/ethers-v6@0.5 typechain@8
```

Then, you can generate like so.
```bash
out_dir="typechain-out"
npm run typechain \
  --target=ethers-v6 \
  --out-dir="$out_dir" \
  "$(pwd)/node_modules/@nibiruchain/solidity/abi/*.json" 
```

This ensures contract interactions are type-checked at compile-time, helping reduce the likelihood of runtime errors.

## Usage in Solidity

This package exports interfaces, types, and constants for each [Nibiru-specific
precompiled contract](https://nibiru.fi/docs/evm/precompiles/nibiru.html). To use these precompiles in Solidity, use the exported constant that represents an instance of the precompile. 

For example in "IOracle.sol", we define:
```solidity
IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);
```

This means another contract can use `NIBIRU_ORACLE` to access each method from
the Nibiru Oracle precompile.
```solidity
import '@nibiruchain/solidity/contracts/IOracle.sol';

contract Example {

  function latestRoundData()
    public
    view
    override
    returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
  {
    return NIBIRU_ORACLE.chainLinkLatestRoundData(pair);
  }
}
```

Similarly, the fungible token precompile defines:
```
IFunToken constant FUNTOKEN_PRECOMPILE = IFunToken(FUNTOKEN_PRECOMPILE_ADDRESS);
```
So, you import this in your contract 
```solidity
import '@nibiruchain/solidity/contracts/IFunToken.sol';

// Methods:
// FUNTOKEN_PRECOMPILE.sendToBank
// FUNTOKEN_PRECOMPILE.sendToEvm
// FUNTOKEN_PRECOMPILE.balance
// FUNTOKEN_PRECOMPILE.bankBalance
// FUNTOKEN_PRECOMPILE.bankMsgSend
// FUNTOKEN_PRECOMPILE.whoAmI
```

## Hacking

[Hacking - Nibiru EVM Solidity Embeds](./HACKING.md)
