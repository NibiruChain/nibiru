---
order: 2
title: "Using the Band Protocol Oracle on Nibiru"
description: >-
  A detailed guide for developers using Band Protocol's oracle solution on Nibiru.
---

# Using the Band Protocol Oracle on Nibiru

{{ $frontmatter.description }}

Band Protocol provides decentralized price feeds for various assets on Nibiru.
This guide covers how to integrate and use Band Protocol's oracle solution
effectively.

## Band Oracle Addresses on Nibiru

| Chain          | Address                                                                                                                         |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Nibiru Testnet | [0x8c064bCf7C0DA3B3b090BAbFE8f3323534D84d68](https://evm-explorer.nibiru.fi/address/0x8c064bCf7C0DA3B3b090BAbFE8f3323534D84d68) |
| Nibiru Mainnet | [0x9503d502435f8e228b874Ba0F792301d4401b523](https://nibiscan.io/address/0x9503d502435f8e228b874Ba0F792301d4401b523)            |

## Smart Contracts

[GitHub・Band Standard Reference Contracts (Solidity)](https://github.com/bandprotocol/band-std-reference-contracts-solidity)

## Available Oracle Price Feeds

The provided feeds will be updated at a minimum or the faster of (i) per time
interval of 30 minutes; or (ii) per 3% price deviation or 0.5% price deviation
for stablecoins.

- BTC
- ETH
- DOGE
- BNB
- TRX
- USDT
- USDC
- XRP
- SOL
- SUI

## Related Documentation

- [Integrating with Oracles on Nibiru](./index.md)
- [Nibiru Oracle Overview](../../../ecosystem/oracle/index.md)
- [Oracle Designs](../../../ecosystem/oracle/defi-designs.md)
- [Enhancing Nibiru with Band Oracle](https://blog.bandprotocol.com/enhancing-nibiru-with-band-oracle/)
