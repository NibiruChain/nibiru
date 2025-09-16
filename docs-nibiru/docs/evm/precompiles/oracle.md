---
order: 5
footer:
  newsletter: false
---

# Understanding the Oracle Precompile

In the Nibiru codebase, [IOracle.sol
precompile](https://github.com/NibiruChain/nibiru/blob/main/x/evm/embeds/contracts/IOracle.sol)
provides the contract interface used to interact with Nibiru's Oracle Mechanism.
{synopsis}

## Who This Page is For

::: tip
This page is oriented toward learning and understanding the precompile. Please
visit the ["Nibiru Oracle (EVM)" page in the developer docs](../../dev/evm/oracle.md) if you're looking for guiding examples on
how to **use the oracles** with TypeScript or Solidity.
:::

## Methods of the Oracle Precompile

The precompile is accessible at a fixed address, and it provides two
functions, `IOracle.queryExchangeRate` and `IOracle.chainLinkLatestRoundData`:

```solidity
address constant ORACLE_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000801;

IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);
```

### 1. `NIBIRU_ORACLE.queryExchangeRate`

This function returns the latest exchange rate for a given asset pair.

#### Parameters

- `pair (string memory)`: The asset pair to query. Example values: "ubtc:uusd", the USD price of BTC. "unibi:uusd", the USD price of NIBI.

#### Returns

- `price (uint256)`: The exchange rate for the given pair.
- `blockTimeMs (uint64)`: The timestamp (in milliseconds) when the price was last updated.
- `blockHeight (uint64)`: The block height when the price was last updated.

#### Example Usage

```solidity
IOracle constant NIBIRU_ORACLE = IOracle(ORACLE_PRECOMPILE_ADDRESS);

function getPrice(string memory pair) public view returns (uint256 price) {
    (price,,) = NIBIRU_ORACLE.queryExchangeRate(pair);
}
```

### 2. `NIBIRU_ORACLE.chainLinkLatestRoundData`

Fetches the latest price data from ChainLink for a given asset pair.

#### Parameters

- `pair (string memory)`: The asset pair to query.

#### Returns

- `roundId (uint80)`: The ID of the price round.
- `answer (int256)`: The latest price of the asset pair.
- `startedAt (uint256)`: Timestamp when the round started.
- `updatedAt (uint256)`: Timestamp when the price was last updated.
- `answeredInRound (uint80)`: The round in which the answer was computed.

#### Example Usage

```solidity
function getChainLinkPrice(string memory pair) public view returns (int256 price) {
    (,price,,,) = NIBIRU_ORACLE.chainLinkLatestRoundData(pair);
}
```

## ChainLink Interface

The Oracle Precompile integrates with ChainLink's AggregatorV3Interface, allowing access to decentralized price feeds.

```solidity
interface ChainLinkAggregatorV3Interface {
    function decimals() external view returns (uint8);
    function description() external view returns (string memory);
    function version() external view returns (uint256);
    function getRoundData(uint80 _roundId) external view returns (
        uint80 roundId,
        int256 answer,
        uint256 startedAt,
        uint256 updatedAt,
        uint80 answeredInRound
    );
    function latestRoundData() external view returns (
        uint80 roundId,
        int256 answer,
        uint256 startedAt,
        uint256 updatedAt,
        uint80 answeredInRound
    );
}
```

## Fetching Active Oracle Pairs

The Nibiru Oracle supports multiple asset pairs, which can be queried dynamically. The currently active markets can be retrieved using the Nibiru CLI with the following command:

```bash
nibid query oracle actives | jq
```

### Example Response

```json
{
  "actives": [
    "uatom:uusd",
    "ubtc:uusd",
    "ueth:uusd",
    "unibi:uusd",
    "uusdc:uusd",
    "uusdt:uusd"
  ]
}
```

Any of these asset pairs can be used as input to the `queryExchangeRate` function of the Oracle Precompile. If the pair is valid, the function returns the exchange rate; otherwise, it returns an error.

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
