# Recollateralize

**Recollateralize** is a function that incentivizes the caller to add up to the amount of collateral needed to reach some **target collateral ratio** (`collRatioTarget`). Recollateralize checks if the USD value of collateral in the protocol is below the required amount defined by the current collateral ratio.

## Concepts

### How much collateral is needed to reach a certain `collRatio`?

Suppose an amount `supplyUSDM` of USDM is in circulation at $1 at some inital collateral ratio, `collRatioStart`. The total USD value of the collateral in Matrix is denoted `collUSDVal`. If USDM falls in price below the lower band, the collateral ratio will increase to `collRatioTarget`, which is the target ratio.  

In order to reach the target `collRatioTarget` with a constant `supplyUSDM`, more collateral needs to be added to the system. This amount can be given by:
```go
collUSDValEnd := supplyUSDM * collRatioEnd
collNeeded := collUSDValEnd - collUSDVal
```

### Incentives for the caller of `Recollateralize`

The caller is given bonus MTRX for bringing the value of the protocol's collateral up to the appropriate value as defined by `collRatioTarget`. This bonus rate is some percentage of the collateral value provided.

Let:
- `collNeeded` (sdk.Int): Amount of collateral needed to reach the target `collRatio`.
- `priceColl` (sdk.Dec): USD price of the collateral  
- `priceMTRX` (sdk.Dec): USD price of MTRX.
- `bonusRate` (sdk.Dec): Defaults to 0.2% (20 bps). The bonus rate gives the caller an incentive to recollateralize Matrix to the target `collRatioTarget`.

Thus, the caller receives an amount of MTRX, `mtrxOut`:
```go
mtrxOut * priceMTRX = (collNeeded * priceColl) * (1 + bonusRate)
mtrxOut = (collNeeded * priceColl) * (1 + bonusRate) / priceMTRX
```

#### References: 
- https://github.com/MatrixDao/matrix/issues/118
