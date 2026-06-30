---
order: 1
title: "Integrating with Oracles on Nibiru"
description: >
  A guide for developers integrating oracle solutions with Nibiru. Nibiru
  supports multiple oracle solutions to provide reliable price feeds and external
  data to your applications. This guide covers the available oracle integrations
  and how to use them effectively.
---

# Integrating with Oracles on Nibiru

{{ $frontmatter.description }}

## [Nibiru Oracle (EVM) - Usage Guide](../../evm/oracle.md)

The native Nibiru Oracle provides exchange rate data and ChainLink-like oracle smart contracts through a precompile, allowing developers to query real-time price information directly on-chain.

See: [ChainLink Aggregator Feeds - Nibiru Mainnet](../../evm/oracle.md#chainlink-like-feeds-nibiru-mainnet)

*All oracle contracts listed here are deployed with 8 decimals*. 

| Token | Name | Address |
| --- | --- | --- |
| **ETH** | Ethereum   | [0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c](https://nibiscan.io/address/0x63b8426F71C3eDbF15A55EeA4915625892Ea9A4c/contract/6900/code) |
| **NIBI** | Nibiru   | [0xb15F7a4b9AD2db05D91f06df9eA7D56EBe8e6B27](https://nibiscan.io/address/0xb15F7a4b9AD2db05D91f06df9eA7D56EBe8e6B27/contract/6900/code) |
| **stNIBI** | [Liquid Staked NIBI](https://nibiru.fi/ecosystem/apps/liquid-staked-nibiru-stnibi)    | [0x2889206C6eDAfD5de621B5EA3999B449879edC70](https://nibiscan.io/address/0x2889206C6eDAfD5de621B5EA3999B449879edC70/contract/6900/code) |
| **BTC** | Bitcoin   | [0xc8FD30cA96B6D120Fc7646108E11c13E8bb128Eb](https://nibiscan.io/address/0xc8FD30cA96B6D120Fc7646108E11c13E8bb128Eb/contract/6900/code) |
| **USDC** | Circle   | [0xBecDA6de445178B3D45aa710F5fB09F72E3e1340](https://nibiscan.io/address/0xBecDA6de445178B3D45aa710F5fB09F72E3e1340/contract/6900/code) |
| **USDT** | Tether   | [0x86C6814Aa44fA22f7B9e0FCEC6F9de6012F322f8](https://nibiscan.io/address/0x86C6814Aa44fA22f7B9e0FCEC6F9de6012F322f8/contract/6900/code) |
| **uBTC** | uBTC from B Squared Network    | [0x1BCA696B83D6d6D67398f20C530aAC8033B53dF2](https://nibiscan.io/address/0x1BCA696B83D6d6D67398f20C530aAC8033B53dF2/contract/6900/code) |
| **sUSDa** | Yield-bearing USDa (Avalon Labs) | [0x4B13Cb07F975aEe89448258babb482378ddA4C32](https://nibiscan.io/address/0x4B13Cb07F975aEe89448258babb482378ddA4C32/contract/6900/code) |

## Band Protocol Oracle

[Using the Band Protocol Oracle on Nibiru](./band-oracle.md)

Band Protocol provides decentralized price feeds for various assets, offering high reliability and frequent updates.

## Supra Pull Oracle

1. Storage

   - 0x2693Aa16Fe576b9cF5F05BF1F4513d51552Cec04

2. Pull
   - 0xbCD5391A146Bacc6d5cE170C1EB1eE52C3d46Ff2

## Related Documentation

1. Oracle Integration Guides

   - [Nibiru Oracle (EVM) - Usage Guide](../../evm/oracle.md)
   - [Using the Band Protocol Oracle on Nibiru](./band-oracle.md)

2. Concept Docs on the Nibiru Oracle
   - [Nibiru Oracle Overview](../../../ecosystem/oracle/index.md)
   - [Oracle Designs](../../../ecosystem/oracle/defi-designs.md)
