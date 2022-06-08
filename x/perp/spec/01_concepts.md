# Concepts                    <!-- omit in toc -->

- [Perp Positions](#perp-positions)
  - [Mark Price and Index Price](#mark-price-and-index-price)
  - [Leverage and Perp Position Value](#leverage-and-perp-position-value)
  - [Margin and Margin Ratio](#margin-and-margin-ratio)
  - [Profits and Losses (PnL)](#profits-and-losses-pnl)
  - [Funding Payments](#funding-payments)
- [Virtual Pools](#virtual-pools)
- [Liquidations](#liquidations)
    - [Liquidation event](#liquidation-event)
- [References](#references)

---

# Perp Positions

- TODO explanation of perpetual futurues contract as a derivative
- TODO what is a perp

## Mark Price and Index Price

#### Mark Price

The **mark price** is the value of the derivative asset (the perp) on the exchange. Mark price is used to calculate **profits and losses (PnL)** and determines whether a position has enough collateral backing it to stay "above water" or if it should be liquidated. The term "mark price" gets its name from the fact that it describes a position's **mark-to-market PnL**, the profit or loss to be realized over the contract period based on current market conditions (perps exchange price).

#### Index Price

The value of a perp's underlying asset is referred to as the **index price**. For example, a BTC:USD perp has BTC as its **base asset** and dollar collateral such as USDC as could be its **quote asset**. The dollar value of BTC on spot exchanges is the index price of the BTC:USD perp. Thus we'd call BTC **"the underlying"**. Usually, the index price is taken as the average of spot prices across major exchanges. 

## Leverage and Perp Position Value

#### Position Size

Suppose a trader wanted exposure to 5 ETH through the purchase of a perpetual contract. On Nibi-Perps, going long on 5 ETH means that the trader buys the ETH perp with a **positiion size** of 5.

#### Position Notional Value

The notional value of the position, or **position notional**, is the total value a position controls  in units of the quote asset. Notional value expresses the value a derivatives contract theoretically controls. On Nibiru, it is defined more concretely by

```
leverage = positionNotional / margin.
```

Let's say that the index price of ether is $3000 in our previous example. This implies that the trader with a long position of size 5 has a position notional of $15,000. And if the trader has 10x **leverage**, for example, she must have put down $1000 as margin (collateral backing the position).  

## Margin and Margin Ratio

**Margin** is the amount of collateral used to back a position. Margin is expressed in units of the quote asset. At genesis, Nibi-Perps uses USDC as the primary quote asset. 

The margin ratio is defined by:

```
marginRatio = (margin + unrealizedPnL) / positionNotional
```

Here, `unrealizedPnL` is computed using either the mark price or the 15 minute TWAP of mark price; the higher of the two values is used when evaluating liquidation conditions.

When the virtual price is not within the spread tolerance to the index price, the margin ratio used is the highest value between a calculation with the index price (oracle based on underlying) and the mark price (derivative price).

Another good way to think about margin ratio is as the inverse of a position's effective leverage. I.e. if a trader puts down $100 as margin with 5x leverage, the notional is $500 and the margin ratio is 20%, which is equivalent ot `1 / leverage`.

#### Cross Margin versus Isolated Margin

- In a **cross margin** model, collateral is shared between open positions that use the same settlement currency. All open positions then have a combined margin ratio.
- With an **isolated margin** model, the margin assigned to each open position is considered a separate collateral account. 

**Current implementation**: Nibi-Perps uses isolated margin on each trading pair. This means that excess collateral on one position is not affected by a deficit on another (and vice versa). Positions are siloed in terms of liquidation risks, so an underwater ETH:USD position won't have any effect on an open ATOM:USD position, for instance.

In future upgrade, we'd like to implement a cross margin model and allow traders to select whether to use cross or isolated margin in the trading app. This way, traders could elect to have profits from one position can support losses in another. 

## Profits and Losses (PnL)

- TODO Define profits and losses
- TODO Explain PnL calculation

## Funding Payments

Perpetual contracts rely on a scheduled payment between longs and shorts known as **funding payments**. Funding payments are meant to converge the price between the derivate contract, or perp, and its underlying. As a result, they are scaled based on the difference between the mark price and index price.

- TODO Explain purpose of funding payments
- TODO define funding rate
- TODO define funding payment

---

# Virtual Pools

- TODO describe virtual reserves
- TODO describe constant-product curve
- TODO describe slippage
- TODO Explain base ammount limit

---

# Liquidations

**Liquidate** is a function which closes a position and distributes assets based on a liquidation fee that goes to the liquidator and ecosystem fund. Liquidations prevent traders' accounts from falling into negative equity.

A liquidation happens when a trader can no longer meet the margin requirement of their leveraged position. In Nibiru, meeting the margin requirement means maintaining a margin ratio on the position that exceeds the **maintenance margin ratio** (6.25%), which is the minimum margin ratio that a position can have before being liquidated.

### Liquidation event

The liquidation consists of opening a reverse position. If the position was a Sell with a size of 20, we just execute a Buy order for the same amount.

From there, we compute a margin ratio using only the spot price. This margin ratio will be used to define wether bad debt will be created if the margin is below the liquidation fee.

When this margin ratio is higher than the liqudiation fee, we close the position and transfer the fees. Half the fees are sent to the liquidator, and half are sent to the ecosystem fund.

- Otherwise, we compute the margin ratio including the funding payments in the unrealizedPnL to see if we can still pay the liquidation fee
  - If we can pay the liquidation fee to the liquidator, we send the remaining margin to the ecosystem fund
  - Otherwise, we count is as bad debt added onto the position potentially withdraw from the ecosystemFund to the pre-paid vault to pay the trader.

---

# References

- Index Price and Mark Price. BTSE. [[support.btse.com]](https://support.btse.com/en/support/solutions/articles/43000557589-index-price-and-mark-price)
- Notional Value vs. Market Value: An Overview. Investopedia. [[investopedia.com]](https://www.investopedia.com/ask/answers/050615/what-difference-between-notional-value-and-market-value.asp)
- Differences Between Isolated Margin and Cross Margin - Binance. [[binance.com]](https://www.binance.com/en/support/faq/b4e9e6ad70934bd082e8e09e33e69513)
- Isolated and Cross Margin - BitMex. [[bitmex.com]](https://www.bitmex.com/app/isolatedMargin)