---
order: 5
---

# Implementation of the NIBI Tokenomics 

Nibiru's Inflation Module implements the emissions for the [Nibiru (NIBI)
token](../../learn/nibi.md). These emissions go to stakers, the Nibiru Foundation
treasury, and the community reserve. {synopsis}

## Introduction

The Nibiru blockchain's main, native digital asset is called NIBI. As of Nibiru
v1.3.0, NIBI tokens are only ever minted by the Inflation Module, which is a
drop-in replacement for the [Mint Module](./cosmos-sdk/mint.md) from the
Cosmos-SDK. We use a custom implementation in order to properly implement the
Nibiru token economics and release schedule.

## Total Supply

NIBI tokens are not automatically burnt by this module, and the fully diluted
supply is capped at 1.5 billion NIBI if there are no "buyback and burn" events.

## Module Parameters

::: tip
The [NibiruChain/tokenomics GitHub repo](https://github.com/NibiruChain/tokenomics) shows how the coefficients for the following module parameters are computed.
:::

| Parameter            | Description |
|---------------------------|-------------|
| `inflation_enabled`       | The parameter that enables inflation and halts increasing the skipped_epochs. |
| `polynomial_factors`      | Factors of a polynomial in decreasing order. These numbers are used to calculate polynomial inflation. |
| `inflation_distribution`  | Distribution of percentages describing where to send newly minted tokens. |
| `epochs_per_period`       | The number of epochs that must pass before a new period is created. |
| `periods_per_year`        | The number of periods that occur in a year. |
| `max_period`              | The maximum number of periods that have inflation being paid off. After this period, inflation will be disabled. |
| `has_inflation_started`   | Indicates if inflation has started. It's set to false at the start, and stays at true when we toggle inflation on. It's used to track the number of skipped epochs. |

The token emissions are derived from a target rate of increase in each epoch. To
represent these values precisely onchain in discrete chunks, we use a polynomial
since a small-order polynomial can be used to approximate almost any smooth, monotonic curve.

You can query these parameters on mainnet using the [Nibiru CLI](../../dev/cli/README.md).
```bash
# Mainnet CLI Config
nibid config node https://rpc.nibiru.fi:443
nibid config chain-id cataclysm-1
nibid config broadcast-mode sync 
nibid config
```

```bash
nibid query inflation params
```

```js
{
  "inflation_enabled": true,
  "polynomial_factors": [
    "-0.000147085524000000",
    "0.074291982762000000",
    "-18.867415611180000000",
    "3128.641926954698000000",
    "-334834.740631598223000000",
    "17827464.906540066004000000"
  ],
  "inflation_distribution": {
    "staking_rewards": "0.281250000000000000",
    "community_pool": "0.354825000000000000",
    "strategic_reserves": "0.363925000000000000"
  },
  "epochs_per_period": "30",
  "periods_per_year": "12",
  "max_period": "96",
  "has_inflation_started": true
}
```

## Community Reserve

The term "community reserve" in the Nibiru ecosystem refers to the "Community
Pool"  module account, which is designated to hold funds for use in
community-driven projects. These funds are managed through a governance process,
allowing stakeholders to propose and vote on their allocation. 

The reserve receives a portion of newly minted tokens, intended to support
initiatives that may include development grants, educational efforts, or other
activities that contribute to the network. This setup provides a mechanism for
the community to decide on resource allocation to projects aligned with its
interests and goals.

## Related Pages

- [Tokenomics | Nibiru (NIBI) Token Unlocks and Vesting](../../learn/tokenomics.md)
- [Nibiru (NIBI) Token](../../learn/nibi.md)
- [Advanced Topics (Nibiru Architecture)](./index.md)

<!-- TODO: docs: Epochs Module. --> 
<!-- Nibiru divides time into epochs, specified by the Epochs Module. --> 

<!-- TODO: docs: inflation module.

Answer the following for this page.
- [ ] What is max period used for?

--> 
