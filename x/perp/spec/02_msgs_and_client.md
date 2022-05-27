# Messages and Client                    <!-- omit in toc -->

This page describes the message (`Msg`) structures and expected state transitions that these messages bring about when wrapped in transactions. These descriptions are accompanied by documentation for their corresponding CLI commands.

- [OpenPosition](#openposition)
- [ClosePosition](#closeposition)
- [AddMargin](#addmargin)
- [RemoveMargin](#removemargin)
- [Liquidate](#liquidate)

## OpenPosition

`OpenPosition` defines a method for opening or altering a new position, which sends funds the vault to the trader, realizing any outstanding profits and losses (PnL), funding payments, and bad debt.

#### `OpenPosition` CLI command:
// TODO: 
```sh
nibid tx perp open-perp --vpool --side --margin --leverage --base-limit 
```

This command has several required flags:
- `vpool`: Identifier for the position's virtual pool.
- `side`: Either "long" or "short"
- `margin`: The amount of collateral input to back the position. This collateral is the quote asset of the 'vpool'.
- `leverage`:  A decimal number between 1 and 10 (inclusive) that specifies how much leverage the trader wishes to take on.
- `base-limit`: Limiter to ensure the trader doesn't get screwed by slippage.


## ClosePosition

`ClosePosition` defines a method for closing a trader's position, which sends funds the vault to the trader, realizing any outstanding profits and losses (PnL), funding payments, and bad debt.

#### `ClosePosition` CLI command:

```sh
nibid tx perp close-perp [vpool] 
```

## AddMargin

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

## RemoveMargin

`RemoveMargin` further leverages a trader's position by removing some of the margin that backs it without altering its notional value. Removing margin decreases the margin ratio of the position and increases the risk of liquidation. 

#### `RemoveMargin` CLI command:

```sh
nibid tx perp remove-margin [vpool] [margin]
# example
nibid tx perp remove-margin atom:nusd 100nusd
```

## Liquidate

```sh
nibid tx perp liquidate [vpool] [trader]
```