---
order: 5
footer:
  newsletter: false
description: >-
  The IOracle precompile at address 0x0801 returns Sai oracle prices to EVM
  contracts that call the legacy Chainlink-like interface.
---

# Understanding the Oracle Precompile

{{ $frontmatter.description }}

## Who This Page is For

::: tip
This page is oriented toward learning and understanding the precompile. Please
visit the ["Nibiru Oracle (EVM)" page in the developer docs](../../dev/evm/oracle.md) if you're looking for guiding examples on
how to **use the oracles** with TypeScript or Solidity.
:::

## Where the Prices Come From

The precompile is read-only. It does not read native `x/oracle` module state,
which is deprecated and disabled. Each call queries the Wasm adapter configured
under plugin name `x-oracle` in the EVM params field `wasm_plugins`. The adapter
reads the Sai oracle contract.

```text
EVM call to 0x0801 -> EVM params plugin "x-oracle" -> x-oracle Wasm adapter -> Sai oracle contract
```

The precompile supports these pairs: `ubtc:uusd`, `ueth:uusd`, `uatom:uusd`,
`unibi:uusd`, and `uusdc:uusd`. Other pairs, including `uusdt:uusd`, return an
error. Prices have 18 decimals.

## Methods of the Oracle Precompile

The precompile is accessible at a fixed address, and it provides two
functions, `IOracle.queryExchangeRate` and `IOracle.chainLinkLatestRoundData`:

```solidity
address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;

IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);
```

### 1. `NIBIRU_ORACLE.queryExchangeRate`

This function returns the latest Sai oracle price for a given asset pair.

#### Parameters

- `pair (string memory)`: The asset pair to query. Example values: "ubtc:uusd", the USD price of BTC. "unibi:uusd", the USD price of NIBI.

#### Returns

- `price (uint256)`: The price for the given pair, with 18 decimals.
- `blockTimeMs (uint64)`: The time in milliseconds when the Sai oracle last updated the price.
- `blockHeight (uint64)`: The block height of the query, not of the price update.

#### Example Usage

```solidity
IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);

function getPrice(string memory pair) public view returns (uint256 price) {
    (price,,) = NIBIRU_ORACLE.queryExchangeRate(pair);
}
```

### 2. `NIBIRU_ORACLE.chainLinkLatestRoundData`

Returns the latest price for a given asset pair in the Chainlink `latestRoundData` shape.

#### Parameters

- `pair (string memory)`: The asset pair to query.

#### Returns

- `roundId (uint80)`: The block height of the query.
- `answer (int256)`: The latest price of the asset pair, with 18 decimals.
- `startedAt (uint256)`: The Sai oracle update time in seconds.
- `updatedAt (uint256)`: The same Sai oracle update time in seconds. Check it for staleness.
- `answeredInRound (uint80)`: A constant placeholder. Don't rely on it.

#### Example Usage

```solidity
function getChainLinkPrice(string memory pair) public view returns (int256 price) {
    (,price,,,) = NIBIRU_ORACLE.chainLinkLatestRoundData(pair);
}
```

## Full Contract Example

Below is a complete smart contract that interacts with the Nibiru EVM Oracle Precompile:

```solidity
// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

interface IOracle {
    function queryExchangeRate(string memory pair) external view returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight);
    function chainLinkLatestRoundData(string memory pair) external view returns (
        uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound
    );
}

contract OracleTest {
    address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;
    IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);

    function getExchangeRate(string memory pair) public view returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight) {
        return NIBIRU_ORACLE.queryExchangeRate(pair);
    }

    function getChainLinkPrice(string memory pair) public view returns (int256 price, uint80 roundId, uint256 updatedAt) {
        (roundId, price,, updatedAt,) = NIBIRU_ORACLE.chainLinkLatestRoundData(pair);
    }
}
```

## Related Pages

1. ["Nibiru Oracle (EVM)" Developer Docs](../../dev/evm/oracle.md)
2. [`@nibiruchain/evm-core` package on npm](../../dev/evm/npm-evm-core.md)
2. [`@nibiruchain/solidity` package on npm](../../dev/evm/npm-solidity.md)
