# Messages and Client                    <!-- omit in toc -->

This page describes the message (`Msg`) structures and expected state transitions that these messages bring about when wrapped in transactions. These descriptions are accompanied by documentation for their corresponding CLI commands.

- [OpenPosition](#openposition)
- [ClosePosition](#closeposition)
- [AddMargin](#addmargin)
- [RemoveMargin](#removemargin)
- [Liquidate](#liquidate)

---

# OpenPosition

`OpenPosition` defines a method for opening or altering a new position, which sends funds the vault to the trader, realizing any outstanding profits and losses (PnL), funding payments, and bad debt.

#### `OpenPosition` CLI command:

```sh
nibid tx perp open-perp --vpool --side --margin --leverage --base-limit 
```

| Flag | Description | 
| ---  | -------     |
| `vpool` | Identifier for the position's virtual pool.
| `side` |  Either "long" or "short" |
| `margin` | The amount of collateral input to back the position. This collateral is the quote asset of the 'vpool'. |
| `leverage` | The leverage of the new position. Leverage is the ratio between the notional value of the position and the margin-collateral that backs it. A \$500 position with \$100 of margin backing has 5x leverage. Note that the effective leverage of a position is inverse of the position's current margin ratio. |
| `base-limit` | Limiter to ensure the trader doesn't get screwed by slippage. |

# ClosePosition

`ClosePosition` defines a method for closing a trader's position, which sends funds the vault to the trader, realizing any outstanding profits and losses (PnL), funding payments, and bad debt.

#### `ClosePosition` CLI command:

```sh
nibid tx perp close-perp [vpool] 
```

| Flag | Description | 
| ---  | -------     |
| `vpool` | Identifier for the position's virtual pool.

# AddMargin

`AddMargin` deleverages a trader's position by adding margin to it without altering its notional value. Adding margin increases the margin ratio of the position. 

```go
type MsgAddMargin struct {
  // Sender: sdk.AccAddress of the owner of the position
  Sender    string   
  // TokenPair: identifier for the position's virtual pool
  TokenPair string   
  // Margin: Amount of margin (quote units) to add to the position
  Margin    sdk.Coin 
}
```

#### `AddMargin` CLI command:

```sh
nibid tx perp add-margin [vpool] [margin]
```

# RemoveMargin

`RemoveMargin` further leverages a trader's position by removing some of the margin that backs it without altering its notional value. Removing margin decreases the margin ratio of the position and increases the risk of liquidation. 

#### `RemoveMargin` CLI command:

```sh
nibid tx perp remove-margin [vpool] [margin]
# example
nibid tx perp remove-margin atom:nusd 100nusd
```

| Flag | Description | 
| ---  | -------     |
| `vpool` | Identifier for the position's virtual pool.
| `margin` | Integer amount of margin to remove from the position. Recall that margin is in units of the quote asset of the virtual pool.  |

# Liquidate

`Liquidate` is a transaction that allows the caller to fully or partially liquidate an existing position. 

<!-- TODO extend liquidate description -->

#### `Liquidate` CLI command:

```sh
nibid tx perp liquidate [vpool] [trader]
```

| Flag | Description | 
| ---  | -------     |
| `vpool` | Identifier for the position's virtual pool.
| `trader` | sdk.AccAddress of the owner of the position that will be liquidated. |
