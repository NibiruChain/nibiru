---
order: 0
title: "Tokens on Nibiru | Fungible Token Standards - Nibiru"
description: >-
  Nibiru supports fungible tokens as ERC20 tokens in Nibiru EVM and Bank Coins
  in the Bank module. FunToken mappings connect both representations so assets
  can move across Nibiru's virtual machines.
---

# Tokens on Nibiru

{{ $frontmatter.description }}

Nibiru runs Wasm and EVM execution environments side by side. This gives assets
two main token representations:

- **ERC20 tokens** are EVM smart contracts. An ERC20 token is identified by its
  contract address and works with Solidity applications, EVM wallets, and
  Ethereum tooling.
- **Bank Coins** are native Nibiru assets. A Bank Coin is identified by a bank
  denomination, or `denom`, and works with staking, governance, IBC, CosmWasm
  contracts, and interchain wallets.
- **FunToken mappings** connect a Bank Coin denomination with an ERC20 contract.
  A mapped asset can move between Bank Coin and ERC20 form without creating a
  separate wrapped token for each environment.

## Standards

| Standard | Identifier | Primary environment | Learn more |
| --- | --- | --- | --- |
| ERC20 token | EVM contract address | Nibiru EVM | [ERC20 Tokens](./erc20.md) |
| Bank Coin | Bank denomination, for example `unibi` | Bank module, Wasm, IBC | [Bank Coins](./bank-coins.md) |
| FunToken mapping | Bank denomination and ERC20 contract pair | Cross-VM asset movement | [FunToken Mechanism](../../evm/funtoken.md) |

## Where to find token addresses

This section explains token standards. For user-facing token pages and contract
addresses, see [Tokens of Nibiru](../../use/tokens/index.md). For the canonical
registry used by Nibiru software, see the
[Nibiru token registry](https://github.com/NibiruChain/nibiru/tree/main/token-registry).
