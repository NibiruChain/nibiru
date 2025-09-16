---
order: 4
metaTitle: "Nibiru Oracle (EVM)"
title: "Nibiru Oracle (EVM)"
---

# Nibiru Oracle (EVM) - Usage Guide

Nibiru EVM includes a precompile for Nibiru's oracle module, providing access to
exchange rate data and ChainLink-like oracle smart contracts. This allows
developers to query real-time price information for various markets directly
on-chain. {synopsis}

| Section | Purpose |
| --- | --- | 
| [ChainLink-like Feeds - Nibiru Mainnet](#chainlink-like-feeds-nibiru-mainnet) | Go here to find the deployed ChainLink-like oracle contract addresses. |
| [Using the ChainLink-like Nibiru Oracles](#using-the-chainlink-like-nibiru-oracles) | Start here to learn how to use ChainLike-like oracles in your Web3 app or smart contracts |
| [Example: TypeScript, Ethers v6, and the Oracle Precompile](#example-typescript-ethers-v6-and-the-oracle-precompile) | To use the Nibiru EVM Oracle precompile  |
| [Appendix](#appendix) | [Related Pages](#related-pages), and [ChainLink-like Feeds on Nibiru Testnet](#chainlink-like-feeds-nibiru-testnet) |

## ChainLink-like Feeds - Nibiru Mainnet

**Smart Contract Implementation**: These feeds are deployments of
[NibiruOracleChainLinkLike.sol](https://github.com/NibiruChain/nibiru/blob/main/x/evm/embeds/contracts/NibiruOracleChainLinkLike.sol),
a contract that sources data from the Nibiru Oracle Mechanism and exposes the
data in the ChainLink `AggregatorV3Interface` format.

Reference: 
- [Nibiru Networks and RPCs](../networks/README.md)
- [ChainLink-like Feeds - Nibiru Testnet](#chainlink-like-feeds-nibiru-testnet)

| Asset | Decimals, Name | Address |
| --- | --- | --- |
| ETH | 8, Ethereum | 0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c |
| NIBI | 8, Nibiru | 0xb15F7a4b9AD2db05D91f06df9eA7D56EBe8e6B27 |
| stNIBI | 8, Liquid Staked NIBI | 0x2889206C6eDAfD5de621B5EA3999B449879edC70 |
| BTC | 8, Bitcoin | 0xc8FD30cA96B6D120Fc7646108E11c13E8bb128Eb |
| USDC | 8, Circle | 0xBecDA6de445178B3D45aa710F5fB09F72E3e1340 |
| USDT | 8, Tether | 0x86C6814Aa44fA22f7B9e0FCEC6F9de6012F322f8 |

<!-- 

2025-06-23:
The USDT deployment needs to be updated to the new bytecode that has additional
methods for Aave, since those are used by LayerBank.

Older deployments:
| USDC | 8, Circle | 0x22A1eBBe1282d9E4EC64FedF71826E1faD056Eb1 | 

-->

## Using the ChainLink-like Nibiru Oracles

The Oracle precompile integrates with ChainLink's AggregatorV3Interface, allowing
access to decentralized price feeds.

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

### ChainLink-Like Nibiru Oracles in a Web3 App

To query these oracles in a Web3 application, you can import the [`@nibiruchain/evm-core` package on npm](./npm-evm-core.md).

```bash
npm i @nibiruchain/evm-core@latest ethers@6
```

Note that in this example, we use the address for ETH. We include the list of all active
ChainLink-like feeds  in the ["Nibiru Mainnet - ChainLink-like Feeds" section](#nibiru-mainnet-chainlink-like-feeds).

```js
import { ethers } from "ethers";
import { chainlinkLike } from "@nibiruchain/evm-core/ethers";

const doReadChainlink = async (wallet: ethers.ContractRunner) => {
  const oracleAddr = "0x6187b99Cad5fbF30f98A06850d301cb1b31b27b2";
  const oracleCaller = chainlinkLike(wallet, oracleAddr);
  const description = await oracleCaller.description();
  const decimals = await oracleCaller.decimals();
  const out = await oracleCaller.latestRoundData();
  const [roundId, answer, startedAt, updatedAt, answeredInRound] = out;
  console.debug("DEBUG %o: ", {
    out,
    oracleAddr,
    description,
    decimals,
    outCleaner: { roundId, answer, startedAt, updatedAt, answeredInRound },
  });
};
```

Example output - Oracle for ETH on mainnet

```js
DEBUG {
  out: [ 18859259n, 2705840000n, 1739818322n, 1739818322n, 420n ],
  oracleAddr: "0x6187b99Cad5fbF30f98A06850d301cb1b31b27b2",
  description: "Nibiru Oracle ChainLink-like price feed for ueth:uusd",
  decimals: 6n,
  outCleaner: {
    roundId: 18859259n,
    answer: 2705840000n,
    startedAt: 1739818322n,
    updatedAt: 1739818322n,
    answeredInRound: 420n,
  },
}:
```

### ChainLink-Like Nibiru Oracles in Solidity

```bash
npm i @nibiruchain/solidity
```

## Example: TypeScript, Ethers v6, and the Oracle Precompile

To communicate directly with the Nibiru EVM Oracle precompile, you can use the ethers.js library to interact with the contract. The script you provided already demonstrates how to query the Oracle for exchange rates and Chainlink prices using the correct contract address and ABI.

Here's a breakdown of the necessary steps to ensure a smooth communication with the precompile, along with some refinements:

```javascript
const { ethers } = require("hardhat");

async function main() {
  const provider = new ethers.JsonRpcProvider(
    "https://evm-rpc.testnet-2.nibiru.fi/" // Nibiru Testnet RPC URL
  );
  const precompileAddress = "0x0000000000000000000000000000000000000801"; // Oracle Precompile Address

  // Define the IOracle ABI for querying the exchange rate and ChainLink data
  const abi = [
    "function queryExchangeRate(string pair) view returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight)",
    "function chainLinkLatestRoundData(string pair) view returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)",
  ];

  // Connect to the Oracle precompile contract using the ABI and provider
  const oracle = new ethers.Contract(precompileAddress, abi, provider);

  // Define asset pairs to query the exchange rates
  const pairs = [
    "uatom:uusd",
    "ubtc:uusd",
    "ueth:uusd",
    "uusdc:uusd",
    "uusdt:uusd",
  ];

  // Loop through the pairs and query the Oracle for exchange rates and ChainLink prices
  for (const pair of pairs) {
    try {
      // Query the exchange rate for the asset pair
      console.log(`Querying exchange rate for ${pair}...`);
      const [price, blockTimeMs, blockHeight] = await oracle.queryExchangeRate(pair);
      console.log(`Pair: ${pair}`);
      console.log(`Exchange Rate: ${ethers.formatUnits(price, 18)}`); // Format the price to 18 decimals
      console.log(`Block Time (ms): ${blockTimeMs.toString()}`);
      console.log(`Block Height: ${blockHeight.toString()}`);
    } catch (error) {
      console.error(`Error querying exchange rate for ${pair}:`, error);
    }

    try {
      // Query the ChainLink price for the asset pair
      console.log(`Querying ChainLink price for ${pair}...`);
      const [roundId, answer, startedAt, updatedAt, answeredInRound] = await oracle.chainLinkLatestRoundData(pair);

      console.log(`Pair: ${pair}`);
      console.log(`ChainLink Price: ${ethers.formatUnits(answer, 18)}`); // Format the price to 18 decimals
      console.log(`Round ID: ${roundId.toString()}`);
      console.log(`Started At: ${startedAt.toString()}`);
      console.log(`Updated At: ${updatedAt.toString()}`);
      console.log(`Answered In Round: ${answeredInRound.toString()}`);
    } catch (error) {
      console.error(`Error querying ChainLink price for ${pair}:`, error);
    }
  }
}

// Execute the script
main().catch(console.error);
```

### How This Script Works

1. The script connects to the Nibiru EVM Oracle precompile using the predefined address and ABI.
2. It queries the `queryExchangeRate` function to get the price, block time, and block height for each asset pair.
3. It also queries the `chainLinkLatestRoundData` function to fetch the latest ChainLink price data.
4. The results are logged to the console for each pair, with appropriate formatting for clarity.

#### Explanation of Key Components

1. Provider Setup: Connects to the Nibiru Testnet via the [RPC endpoint](../networks/README.md) (<https://evm-rpc.testnet-2.nibiru.fi/>) to establish communication with the Ethereum-compatible network.

1. ABI Definition: Specifies the functions available for interaction with the Oracle smart contracts, allowing querying of prices and exchange rates.

1. Supported Asset Pairs: Lists the asset pairs (e.g., "ueth:uusd", "unibi:uusd") supported by Nibiru’s Oracle system for querying price data.

1. Error Handling: Implements robust error management using try-catch statements to handle potential failures gracefully during queries to the Oracle or ChainLink-like feeds.

1. Formatted Output: Provides clear and precise price data formatted to 18 decimals, ensuring readability and ease of interpretation.

This approach directly communicates with the Nibiru Oracle precompile, and you
can extend this to include other queries or add additional functionality as
needed.

---

## Appendix

### Related Pages

1. [Understanding the Oracle Precompile](../../evm/precompiles/oracle.md)
2. [`@nibiruchain/evm-core` package on npm](./npm-evm-core.md)
3. [`@nibiruchain/solidity` package on npm](./npm-solidity.md)

### ChainLink-like Feeds - Nibiru Testnet

Reference: [Nibiru Networks and RPCs](../networks/README.md)

| Asset | Decimals, Name | Address |
| --- | --- | --- |
| BTC:USD | 8, Bitcoin | 0x1CA6d404DB645a88aE7af276f3E2CdF64A153107 |
| ETH:USD | 8, Ethereum | 0xcAF7BaB290E540f607c977a2e95182514Aeb957f |
| NIBI:USD | 8, Nibiru | 0x2b6C81886001E12341b54b14731358EbAD3a83bd |
| USDC:USD | 8, Circle | 0x9fc81892ea19d6d3Db770A632C8D5Dd1889DE075 |
| USDT:USD | 8, Tether | 0xBC3bac25cdf3410D3f2Bd66DC4f156dB49ac92Cb |
