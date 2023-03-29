# `x/stablecoin`        <!-- omit in toc -->

*******

The `stablecoin` module is responsible for minting and burning NUSD, maintenance of NUSD's price stability, and orchestration of Nibiru Protocol's collateral ratio.

- [Docs - NUSD Stablecoin](https://docs.nibiru.fi/ecosystem/nusd-stablecoin.html)

#### Table of Contents

- **[CLI Usage Guide](#cli-usage-guide)**
  - [Minting Stablecoins](#minting-stablecoins)
- **[Concepts](#concepts)**
  - [Recollateralize](#recollateralize): Recollateralize is a function that incentivizes the caller to add up to the amount of collateral needed to reach some **target collateral ratio** (`collRatioTarget`).
  - [Buybacks](#buybacks): A user can call `Buyback` when there's too much collateral in the protocol according to the target collateral ratio. The user swaps NIBI for UST at a 0% transaction fee and the protocol burns the NIBI it buys from the user.
- **Messages and Events**: [description]
- **Keepers and Parameters**: [description]
- **Module Accounts of `x/stablecoin`**: [description]

---

# CLI Usage Guide

## Minting Stablecoins

In a new terminal, run the following command:

```bash
// send a transaction to mint stablecoin
$ nibid tx stablecoin mint 1000validatortoken --from validator --home data/localnet --chain-id localnet

// query the balance
$ nibid q bank balances cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v
```

<!-- # Module Accounts of `x/stablecoin`

Treasury: TODO docs

StableEF: TODO docs -->

# Concepts

## Recollateralize           

**Recollateralize** is a function that incentivizes the caller to add up to the amount of collateral needed to reach some **target collateral ratio** (`collRatioTarget`). Recollateralize checks if the USD value of collateral in the protocol is below the required amount defined by the current collateral ratio. Here, Nibiru's NUSD stablecoin is taken to be the dollar that determines USD value.

**`collRatio`**: The collateral ratio, or 'collRatio' (sdk.Dec), is a value beteween 0 and 1 that determines what proportion of collateral and governance token is used during stablecoin mints and burns.

#### How much collateral is needed to reach a certain `collRatio`?

Suppose an amount `supplyNUSD` of NUSD is in circulation at $1 at some inital collateral ratio, `collRatioStart`. The total USD value of the collateral in Nibiru is denoted `collUSDVal`. If NUSD falls in price below the lower band, the collateral ratio will increase to `collRatioTarget`, which is the target ratio.  

In order to reach the target `collRatioTarget` with a constant `supplyNUSD`, more collateral needs to be added to the system. This amount can be given by:
```go
collUSDValEnd := supplyNUSD * collRatioEnd
collNeeded := collUSDValEnd - collUSDVal
```

#### Incentives for the caller of `Recollateralize`

The caller is given bonus NIBI for bringing the value of the protocol's collateral up to the appropriate value as defined by `collRatioTarget`. This bonus rate is some percentage of the collateral value provided.

Let:
- `collNeeded` (sdk.Int): Amount of collateral needed to reach the target `collRatio`.
- `priceColl` (sdk.Dec): USD price of the collateral  
- `priceNIBI` (sdk.Dec): USD price of NIBI.
- `bonusRate` (sdk.Dec): Defaults to 0.2% (20 bps). The bonus rate gives the caller an incentive to recollateralize Nibiru to the target `collRatioTarget`.

Thus, the caller receives an amount of NIBI, `nibiOut`:
```go
nibiOut * priceNIBI = (collNeeded * priceColl) * (1 + bonusRate)
nibiOut = (collNeeded * priceColl) * (1 + bonusRate) / priceNIBI
```

#### Implementation

See [[collateral_ratio.go]](../keeper/collateral_ratio.go)


#### References: 
- https://github.com/NibiruChain/nibiru/issues/118


## Buybacks

**TLDR**: A user can call `Buyback` when there's too much collateral in the protocol according to the target collateral ratio. The user swaps NIBI for UST at a 0% transaction fee and the protocol burns the NIBI it buys from the user.

**`collRatio`**: The collateral ratio, or `collRatio` (sdk.Dec), is a value beteween 0 and 1 that determines what proportion of collateral and governance token is used during stablecoin mints and burns.

**`liqRatio`**: The liquidity ratio, or `liqRatio` (sdk.Dec), is a the proportion of the circulating NIBI liquidity relvative to the NUSD (stable) value.

#### When is a "buyback" possible?

The protocol has too much collateral. Here, "protocol" refers to the module account of the `x/stablecoin` module, and "too much" refers to the difference between the `collRatio` and `liqRatio`. 

For example, if there's 10M NUSD in circulation, the price of UST collateral is 0.99 NUSD per UST, and the protocol has 5M UST, the `liqRatio` would be (5M * 0.99) / 10M = 0.495.   
Thus, if the collateral ratio, or `collRatio`, is less than 0.495, the an address with sufficient funds can call `Buyback`. 

#### How does a buyback work?

The protocol has an excess of collateral. Buybacks allow users to sell NIBI to the protocol in exchange for NUSD, meaning that Nibiru Chain is effectively buying back its shares. After this transfer, the NIBI purchased by protocol is burned. This raises the value of the NIBI token for all of its hodlers. 

- Unlike `Recollateralize`, there is no bonus rate for this transaction.


#### Related Issues: 
- https://github.com/NibiruChain/nibiru/issues/117
