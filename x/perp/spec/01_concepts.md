# Concepts                    <!-- omit in toc -->

- [Perp Positions](#perp-positions)
  - [Margin and Margin Ratio](#margin-and-margin-ratio)
- [Virtual Pools](#virtual-pools)
- [Liquidations](#liquidations)
    - [Liquidation event](#liquidation-event)

---

# Perp Positions

TODO section wip

## Margin and Margin Ratio

TODO section wip
#### Margin

TODO section wip

#### Margin Ratio

TODO section wip

The margin ratio is defined as:

`marginRatio = (margin + unrealizedPnL) / positionNotional`

- `unrealizedPnL` is usually computed using both Mark Price and 15 minute TWAP of Mark Price; the higher of the two values is used when evaluating liquidations conditions.
- When the virtual price is not within the spread tolerance to the index price, the margin ratio used is the highest value between a calculation with the index price (oracle based) and the mark price (spot).

If this margin ratio is below the maintenance margin ratio defined by governance, we proceed with the liquidation of the position.

---

# Virtual Pools

TODO section wip

---

# Liquidations

**Liquidate** is a function which closes a position and distributes assets based on a liquidation fee that goes to the liquidator and ecosystem fund. Liquidations prevent traders' accounts from falling into negative equity.

A liquidation happens when a trader can no longer meet the margin requirement of their leveraged position. In Nibiru, meeting the margin requirement means maintaining a margin ratio on the position that exceeds the maintenance margin ratio (6.25%).

### Liquidation event

The liquidation consists of opening a reverse position. If the position was a Sell with a size of 20, we just execute a Buy order for the same amount.

From there, we compute a margin ratio using only the spot price. This margin ratio will be used to define wether bad debt will be created if the margin is below the liquidation fee.

When this margin ratio is higher than the liqudiation fee, we close the position and transfer the fees. Half the fees are sent to the liquidator, and half are sent to the ecosystem fund.

- Otherwise, we compute the margin ratio including the funding payments in the unrealizedPnL to see if we can still pay the liquidation fee
  - If we can pay the liquidation fee to the liquidator, we send the remaining margin to the ecosystem fund
  - Otherwise, we count is as bad debt added onto the position potentially withdraw from the ecosystemFund to the pre-paid vault to pay the trader.
