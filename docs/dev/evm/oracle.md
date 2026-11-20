---
order: 4
title: "Chainlink-like Feeds (EVM)"
description: >
  Deployed Chainlink-style price feed contracts on Nibiru EVM. Each wraps the
  Sai oracle precompile and returns a price with 8 decimals.
---

# Chainlink-like Feeds (EVM)

{{ $frontmatter.description }}

Prices come from the Sai oracle. For the supported pairs, units, and the
precompile itself, see [Sai Oracle Data and How to Read It](../sai-oracle.md).

| Section | Purpose |
| --- | --- |
| [ChainLink-like Feeds - Nibiru Mainnet](#chainlink-like-feeds-nibiru-mainnet) | Deployed feed contract addresses. |
| [Using the ChainLink-like Nibiru Oracles](#using-the-chainlink-like-nibiru-oracles) | How to read a feed in a Web3 app or Solidity. |
| [Appendix](#appendix) | Related pages and testnet feeds. |

## ChainLink-like Feeds - Nibiru Mainnet

**Smart Contract Implementation**: These feeds are deployments of
[NibiruOracleChainLinkLike.sol](https://github.com/NibiruChain/nibiru/blob/main/evm/embeds/contracts/NibiruOracleChainLinkLike.sol),
a contract that reads the Sai oracle through precompile `0x0801` and exposes the
price in the ChainLink `AggregatorV3Interface` format.

*All feeds listed here are deployed with 8 decimals.* On 2026-11-19, each of these feeds returned a price with an update time less than 30 seconds old.

| Token | Name | Address |
| --- | --- | --- |
| **ETH** | Ethereum   | [0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c](https://nibiscan.io/address/0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c/contract/6900/code) |
| **NIBI** | Nibiru   | [0xb15F7a4b9AD2db05D91f06df9eA7D56EBe8e6B27](https://nibiscan.io/address/0xb15F7a4b9AD2db05D91f06df9eA7D56EBe8e6B27/contract/6900/code) |
| **BTC** | Bitcoin   | [0xc8FD30cA96B6D120Fc7646108E11c13E8bb128Eb](https://nibiscan.io/address/0xc8FD30cA96B6D120Fc7646108E11c13E8bb128Eb/contract/6900/code) |
| **USDC** | Circle   | [0xBecDA6de445178B3D45aa710F5fB09F72E3e1340](https://nibiscan.io/address/0xBecDA6de445178B3D45aa710F5fB09F72E3e1340/contract/6900/code) |

Feeds for stNIBI, USDT, uBTC, and sUSDa were deployed earlier. Their calls
revert with `unsupported legacy symbol` because the oracle adapter doesn't map
those assets, so they are not listed.

Reference: 
- [Nibiru Networks and RPCs](../networks/index.md)
- [ChainLink-like Feeds - Nibiru Testnet](#chainlink-like-feeds-nibiru-testnet)


<!-- 

2025-06-23:
The USDT deployment needs to be updated to the new bytecode that has additional
methods for Aave, since those are used by LayerBank.

Older deployments:
| USDC | 8, Circle | 0x22A1eBBe1282d9E4EC64FedF71826E1faD056Eb1 | 

-->

## Using the ChainLink-like Nibiru Oracles

Each feed implements the subset of ChainLink's `AggregatorV3Interface` shown here.

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
  const oracleAddr = "0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c";
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

Example output, from `cast call <feed> "latestRoundData()(uint80,int256,uint256,uint256,uint80)"` on the ETH feed:

```text
49953180 [4.995e7]
177869000000 [1.778e11]
1795102556 [1.795e9]
1795102556 [1.795e9]
420
```

The answer `177869000000` has 8 decimals, so the price is 1,778.69 USD. Both
timestamps are the oracle's update time in seconds. Feed `description()` returns
`Nibiru Oracle ChainLink-like price feed for ueth:uusd`, and `decimals()` returns 8.

### ChainLink-Like Nibiru Oracles in Solidity

```bash
npm i @nibiruchain/solidity
```

---

## Appendix

### Related Pages

1. [Sai Oracle Data and How to Read It](../sai-oracle.md)
2. [`@nibiruchain/evm-core` package on npm](./npm-evm-core.md)
3. [`@nibiruchain/solidity` package on npm](./npm-solidity.md)

### ChainLink-like Feeds - Nibiru Testnet

Reference: [Nibiru Networks and RPCs](../networks/index.md)

| Asset | Decimals, Name | Address |
| --- | --- | --- |
| BTC:USD | 8, Bitcoin | 0x1CA6d404DB645a88aE7af276f3E2CdF64A153107 |
| ETH:USD | 8, Ethereum | 0xcAF7BaB290E540f607c977a2e95182514Aeb957f |
| NIBI:USD | 8, Nibiru | 0x2b6C81886001E12341b54b14731358EbAD3a83bd |
| USDC:USD | 8, Circle | 0x9fc81892ea19d6d3Db770A632C8D5Dd1889DE075 |
