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

## Sai Oracle

Nibiru's price data comes from the Sai oracle. [Sai Oracle Data and How to Read It](../../sai-oracle.md) covers reads from EVM contracts, CosmWasm contracts, a terminal, and AI agents. Deployed Chainlink-style feed contracts are on the [Chainlink-like Feeds (EVM)](../../evm/oracle.md) page.

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

   - [Sai Oracle Data and How to Read It](../../sai-oracle.md)
   - [Chainlink-like Feeds (EVM)](../../evm/oracle.md)
   - [Using the Band Protocol Oracle on Nibiru](./band-oracle.md)

2. Concept Docs on the Nibiru Oracle
   - [Nibiru Oracle Overview](../../../ecosystem/oracle/index.md)
   - [Oracle Designs](../../../ecosystem/oracle/defi-designs.md)
