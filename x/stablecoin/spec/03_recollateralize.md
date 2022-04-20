# Recollateralize           <!-- omit in toc -->

**Recollateralize** is a function that incentivizes the caller to add up to the amount of collateral needed to reach some **target collateral ratio** (`collRatioTarget`). Recollateralize checks if the USD value of collateral in the protocol is below the required amount defined by the current collateral ratio. Here, Nibiru's NUSD stablecoin is taken to be the dollar that determines USD value.

- [Concepts](#concepts)
  - [How much collateral is needed to reach a certain `collRatio`?](#how-much-collateral-is-needed-to-reach-a-certain-collratio)
  - [Incentives for the caller of `Recollateralize`](#incentives-for-the-caller-of-recollateralize)
- [Implementation](#implementation)

## Concepts

**`collRatio`**: The collateral ratio, or 'collRatio' (sdk.Dec), is a value beteween 0 and 1 that determines what proportion of collateral and governance token is used during stablecoin mints and burns.

### How much collateral is needed to reach a certain `collRatio`?

Suppose an amount `supplyNUSD` of NUSD is in circulation at $1 at some inital collateral ratio, `collRatioStart`. The total USD value of the collateral in Nibiru is denoted `collUSDVal`. If NUSD falls in price below the lower band, the collateral ratio will increase to `collRatioTarget`, which is the target ratio.  

In order to reach the target `collRatioTarget` with a constant `supplyNUSD`, more collateral needs to be added to the system. This amount can be given by:
```go
collUSDValEnd := supplyNUSD * collRatioEnd
collNeeded := collUSDValEnd - collUSDVal
```

### Incentives for the caller of `Recollateralize`

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

## Implementation

See [[collateral_ratio.go]](../keeper/collateral_ratio.go)


#### References: 
- https://github.com/NibiruChain/nibiru/issues/118
