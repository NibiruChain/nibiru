# `x/perp`                        <!-- omit in toc -->

---

The perp module powers the Nibi-Perps exchange. This module enables traders to open long and short leveraged positions and houses all of the PnL calculation and liquidation logic.

#### Table of Contents

- **Concepts of `x/perp`** - [[01_concepts.md]](spec/01_concepts.md): Specialized concepts and definitions in the module.
- **Messages and Client** - [[02_msgs_and_client.md]](spec/02_msgs_and_client.md): Documentation for all CLI commands and their corresponding message (`Msg`) structures. This section also details the expected state transitions that these messages bring about when wrapped in transactions. 
- **State** - [[03_state.md]](spec/03_state.md): Describes the structures expected to be marshalled into the store and their keys.
- **Events** - [[04_events.md]](spec/04_events.md): Lists and describes the event types used.

## CLI commands

To see the list of query and transaction commands, use:

```bash
nibid tx perp --help
nibid query perp --help
```

CLI code is contained in the `/perp/client/cli` directory.

## Perp Ecosystem Fund (PerpEF) 
<!-- TODO Complete section and move a "Module Accounts" section inside concepts. -->

The PerpEF is a module account on Nibiru Protocol. All of its interactions can be encapsulated in two keeper methods.
- `WithdrawFromPerpEF()`
- `DepositToPerpEF()`


## Queries
<!-- TODO document queries and add to client file. -->


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


