# Contracts <!-- omit in toc -->

- [Core Contracts](#core-contracts)
- [Example Contracts](#example-contracts)
- [Utility Contracts](#utility-contracts)

---

## Core Contracts

- [**core-token-vesting**](./core-token-vesting/README.md)
- [**core-token-vesting-v2**](./core-token-vesting-v2/README.md)
- [**core-compounder**](./core-compounder): Simple contract to allow third parties
  to stake funds without being able to withdraw/unstake them.

## Example Contracts

- [**nibi-stargate**](./nibi-stargate/README.md): Example smart contract that showcases how to use the Nibiru standard (nibiru-std) library to execute `CosmosMsg::Stargate` transactions for the token factory module.

## Utility Contracts

- **lockup**: Smart contract that enables users to lock or bond tokens for arbitrary durations. This contract can be used as a building block in combination with a contract like `incentives` to implement liquidity mining incentives or other yield mechanisms.

- **incentives**: Smart contract for funding lockups based with tokens.
