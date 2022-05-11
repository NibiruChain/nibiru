# Liquidate

**Liquidate** is a function which close a position and distribute the assets to its claimants. It is an event that occur to prevent traders' account from falling into negative equity.

A liquidation process happens when a trader can no longer meet the margin requirement of their leverage position. In our protocol, this is defined by the trader's margin ratio being less than the maintenance margin ratio.


## Concepts

### Margin Ratio

The margin ratio is defined as:

$
\mathit {marginRatio} = \frac {\mathit{margin} + \mathit {unrealizedPnL^*}} {\mathit {positionNotional}}
$

- `unrealizedPnL` is usually computed using both Mark Price and 15 minute TWAP of Mark Price; the higher of the two values is used when evaluating liquidations conditions.
- When the virtual price is not within the spread tolerance to the index price, the margin ratio used is the highest value between a calculation with the index price (oracle based) and the mark price (spot).


If this margin ratio is below the maintenance margin ratio defined by governance, we proceed with the liquidation of the position.

### Liquidation event

The liquidation consist in opening a reverse position. If the position was a Sell with a size of 20, we just execute a Buy order for the same amount.

From there, we compute a margin ratio using only the spot price. This margin ratio will be used to define wether bad debt will be created if the margin is below the liquidation fee.

- When this margin ratio is higher than the liqudiation fee, we close the position and transfer the fees.
    - Half the fees are sent to the liquidator
    - The other half is sent to the insurance fund.

- Otherwise, we compute the margin ratio including the funding payments in the unrealizedPnL to see if we can still pay the liquidation fee
    - If we can pay the liquidation fee to the liquidator, we send the remaining margin to the insurance fund
    - Otherwise, we count is as bad debt added onto the position potentially withdraw from the insuranceFund to the pre-paid vault to pay the trader.
