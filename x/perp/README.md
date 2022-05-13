# Perp Design 

## Perp Data Structures

#### type Position (on a virtual pool)

- `size` (sdk.Dec): Denominated in vpool.base (y)
- `margin` (sdk.Dec): Isolated margin
- `openNotional` (sdk.Dec): Quote asset (x) value of position when opening. The cost of the position.
- `lastUpdatedCumulativePremiumFraction` (sdk.Dec): For calculating funding payment. Recorded every time a trader opens, reduces, or closes a position
- `liquidityHistoryIndex` (sdk.Int): 
- `blockNumber` (sdk.Int): Blocker number of the last position.

#### type LiquidityChangedSnapshot (of a virtual pool)

- `cumulativeNotional` (sdk.Dec):
- `X` (sdk.Dec): Quote assets of the virtual pool just before liquidity changed.
- `Y` (sdk.Dec): Base assets of the virtual pool just before liquidity changed.
- `totalPositionSize` (sdk.Dec): Total position sized owned by the virtual pool after the last snapshot was taken. Equal to `currentBaseReserve - lastLiquidityChangedHistoryItem.baseReserve + prevTotalPositionSize`

## Clearing House

The `ClearingHouse` can take several actions that affect the reserves of a virtual pool. Here's a brief description 
- [ ] `AddMargin(vpool string, trader sdk.AccAddress, margin sdk.Int) -> nil`: Increase the margin ratio of position by adding margin.
- [ ] `RemoveMargin(vpool string, trader sdk.AccAddress, margin sdk.Int) -> nil`: Decrease the margin ratio of position by removing margin.
- [ ] `SettlePosition`
- [ ] `OpenPosition(vpool string, isLong bool, yIn sdk.Int, lev sdk.Dec):`
- [ ] `ClosePosition`
- [ ] `Liquidate(vpool string, trader sdk.AccAddress)`
- [ ] `PayFunding`
- [ ] `UnrealizedPnL(vpool string, trader sdk.AccAddress) -> (pnl sdk.Dec)`

## Virtual Pools (Vpools)

Fields:
- `x sdk.DecCoin`: Quote reserves. `x.Denom` is the identifier for the token.
- `y sdk.DecCoin`: Base reserves. `y.Denom` is the identifier for the token.
- `xAmtLimit`:
- `yAmtLimit`:

Pool creation and annhilation functions:
- `CreatePool`
- `ShutdownPool`

Main functions:
- `SwapQuoteForBase() -> sdk.Dec`
- `SwapBaseForQuote() -> sdk.Dec` 
- `SettleFunding() -> sdk.Dec`

"Get" functions:
- `InputTruePrice() -> (sdk.Dec)`:
- `OutputTruePrice() -> (sdk.Dec)`:
- `InputPrice() -> (sdk.Dec)`:
- `OutputPrice() -> (sdk.Dec)`:
- `InputPriceAtReserves(x, y, inY) -> (sdk.Dec)`: 
- `OutputPriceAtReserves(x, y, outX) -> ( sdk.Dec)`: 

## Perp Ecosystem Fund (PerpEF)

The PerpEF is a module account on Nibiru Protocol. All of its interactions can be encapsulated in two keeper methods.
- `WithdrawFromPerpEF()`
- `DepositToPerpEF()`


## Queries

#### QueryPositionInfo

Given the `vpool` and `trader`, one could query the 
`QueryPositionInfo(vpool string, trader sdk.AccAddress) -> PositionInfo`

```go
// A single trader's position information on a given Vpool.
type PositionInfo struct {
  MarginRatio sdk.Dec
  Position perptypes.Position
}
```

#### QueryAllVpools

`QueryAllVpools() -> []string`: Returns a list of all of the pool names.

#### QueryVpoolPrices

`QueryVpoolPrices() -> map[string]sdk.Dec`: Returns ech virtual pool and its corresponding price.


