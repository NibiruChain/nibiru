<!--
order: 1
-->
# Concepts | x/perp                    <!-- omit in toc -->

- [Perp Positions](#perp-positions)
  - [Mark Price and Index Price](#mark-price-and-index-price)
  - [Leverage and Perp Position Value](#leverage-and-perp-position-value)
  - [Margin and Margin Ratio](#margin-and-margin-ratio)
  - [Funding Payments](#funding-payments)
- [Virtual Pools](#virtual-pools)
- [Liquidations](#liquidations)
- [References](#references)

---

# Perp Positions

A perpetual contract, or perp, is a type of crypto-native derivative that enables traders to speculate on price movements without holding the underlying asset. Nibiru allows traders to trade perps with leverage using stablecoins like USDC as collateral. 

## Mark Price and Index Price

#### Mark Price

The **mark price** is the value of the derivative asset (the perp) on the exchange. Mark price is used to calculate **profits and losses (PnL)** and determines whether a position has enough collateral backing it to stay "above water" or if it should be liquidated. The term "mark price" gets its name from the fact that it describes a position's **mark-to-market PnL**, the profit or loss to be realized over the contract period based on current market conditions (perps exchange price).

#### Index Price

The value of a perp's underlying asset is referred to as the **index price**. For example, a BTC:USD perp has BTC as its **base asset** and dollar collateral such as USDC as could be its **quote asset**. The dollar value of BTC on spot exchanges is the index price of the BTC:USD perp. Thus we'd call BTC **"the underlying"**. Usually, the index price is taken as the average of spot prices across major exchanges. 

## Leverage and Perp Position Value

#### Position Size

Suppose a trader wanted exposure to 5 ETH through the purchase of a perpetual contract. On Nibi-Perps, going long on 5 ETH means that the trader buys the ETH perp with a **position size** of 5. Position size is computed as the position notional mutlipled by the mark price of the asset. 

```go 
k = baseReserves * quoteReserves
notionalDelta = margin * leverage // (leverage is negative if short)
baseReservesAfterSwap = k / (quoteReserves + notionalDelta)
position_size = baseReserves - baseReservesAfterSwap
```

#### Position Notional Value

The notional value of the position, or **position notional**, is the total value a position controls  in units of the quote asset. Notional value expresses the value a derivatives contract theoretically controls. On Nibiru, it is defined more concretely by

```go
positionNotional = abs(quoteReserves - k / (baseReserves + position_size))
leverage = positionNotional / margin.
```

Let's say that the mark price of ether is \$3000 in our previous example. This implies that the trader with a long position of size 5 has a position notional of \$15,000. And if the trader has 10x **leverage**, for example, she must have put down \$1500 as margin (collateral backing the position). 

## Margin and Margin Ratio

**Margin** is the amount of collateral used to back a position. Margin is expressed in units of the quote asset. At genesis, Nibi-Perps uses USDC as the primary quote asset. 

The margin ratio is defined by:

```
marginRatio = (margin + unrealizedPnL) / positionNotional
```

Here, `unrealizedPnL` is computed using either the mark price or the 15 minute TWAP of mark price; the higher of the two values is used when evaluating liquidation conditions.

When the virtual price is not within the spread tolerance to the index price, the margin ratio used is the highest value between a calculation with the index price (oracle based on underlying) and the mark price (derivative price).

Another good way to think about margin ratio is as the inverse of a position's effective leverage. I.e. if a trader puts down $100 as margin with 5x leverage, the notional is \$500 and the margin ratio is 20%, which is equivalent ot `1 / leverage`.

#### Cross Margin versus Isolated Margin

- In a **cross margin** model, collateral is shared between open positions that use the same settlement currency. All open positions then have a combined margin ratio.
- With an **isolated margin** model, the margin assigned to each open position is considered a separate collateral account. 

**Current implementation**: Nibi-Perps uses isolated margin on each trading pair. This means that excess collateral on one position is not affected by a deficit on another (and vice versa). Positions are siloed in terms of liquidation risks, so an underwater ETH:USD position won't have any effect on an open ATOM:USD position, for instance.

In future upgrade, we'd like to implement a cross margin model and allow traders to select whether to use cross or isolated margin in the trading app. This way, traders could elect to have profits from one position support losses in another. 

<!--  

## Profits and Losses (PnL)

- TODO Explain PnL calculation
- TODO Q: When are PnL calculations completed?
-->

## Funding Payments

Perpetual contracts rely on a scheduled payment between longs and shorts known as **funding payments**. Funding payments are meant to converge the price between the derivate contract, or perp, and its underlying. As a result, they are scaled based on the difference between the mark price and index price.

Longs and shorts are paid with the exact funding rate formula [used by FTX](https://help.ftx.com/hc/en-us/articles/360027946571-Funding). Realized and unrealized funding payments are updated every block directly on each position. Global funding calculations are recorded in a time-weighted fashion, where the **funding rate** is the difference between the mark TWAP and index TWAP divided by the number of funding payments per day:

```go
fundingRate = (markTWAP - indexTWAP) / fundingPaymentsPerDay
```

In the initial version of Nibi-Perps, these payments will occur every half-hour, implying a `funding_payments_per_day` value of 48. This setup is analogous to a traditional future that expires once a day. If a perp trades consistently at 2% above its underlying index price, the funding payments would amount to 2% of the position size after a full day.   

If the funding rate is positive, mark price > index price and longs pay shorts. Nibi-Perps automatically deducts the funding payment amount from the margin of the long positions. 

```go
fundingPayment = positionSize * fundingRate
```

Here, position size refers to amount of base asset represented by the derivative. I.e., a BTC:USD perp with 7 BTC of exposure would have a position size of 7.

---

# Virtual Pools

For information on the virtual pools, see the [`x/vpool` specification](../../vpool/README.md).

---

# Liquidations

**Liquidate** is a function which closes a position and distributes assets based on a liquidation fee that goes to the liquidator and ecosystem fund. Liquidations prevent traders' accounts from falling into negative equity.

A liquidation happens when a trader can no longer meet the margin requirement of their leveraged position. In Nibiru, meeting the margin requirement means maintaining a margin ratio on the position that exceeds the **maintenance margin ratio** (6.25%), which is the minimum margin ratio that a position can have before being liquidated.

When a liquidator address sends a message to liquidate a position, the protocol keeper first computes the margin ratio of the position using the mark price. The notional is taken to be that maximum of the `spot_mark` (mark at an instance in time) notional and `TWAP_mark` notional. Similarly, the unrealized PnL is taken to be the max of the `spot_mark` PnL and `TWAP_mark` PnL. This computation realizes any outstanding funding payments on the position, tells us whether or not the position is underwater, and tells us if the position has "**bad debt**" (margin owed in excess of the collateral backing the position).

If this margin ratio is below the maintenance margin ratio, the liquidation message will close the position. This consists of opening a reverse position with a size equivalent to the one that is currently open, which brings the size to zero. A liquidation fee is taken out of the margin and distributed in some split (currently 50:50) between the Nibi-Perps Ecosystem Fund (Perp EF) and the liquidator. If any margin remains in the position after the liquidation fee is taken out, this remaining margin is sent back to the owner of the position. And if bad debt is created by the liquidation fee, it is payed by the Perp EF.

---

# References

- Index Price and Mark Price. BTSE. [[support.btse.com]](https://support.btse.com/en/support/solutions/articles/43000557589-index-price-and-mark-price)
- Notional Value vs. Market Value: An Overview. Investopedia. [[investopedia.com]](https://www.investopedia.com/ask/answers/050615/what-difference-between-notional-value-and-market-value.asp)
- Differences Between Isolated Margin and Cross Margin - Binance. [[binance.com]](https://www.binance.com/en/support/faq/b4e9e6ad70934bd082e8e09e33e69513)
- Isolated and Cross Margin - BitMex. [[bitmex.com]](https://www.bitmex.com/app/isolatedMargin)
- Funding. FTX Crypto Derivatives Exchange. [[help.ftx.com]](https://help.ftx.com/hc/en-us/articles/360027946571-Funding)