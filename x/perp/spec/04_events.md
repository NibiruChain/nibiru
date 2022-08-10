# Events                        <!-- omit in toc -->

Here, we list and describe the event types used in x/perp.

Events in the Cosmos-SDK are Tendermint application blockchain interface (ABCI) events.
These are returned by ABCI methods such as CheckTx, DeliverTx, and Query.

Events allow applications to associate metadata about ABCI method execution with
the transactions and blocks this metadata relates to. Events returned via these
ABCI methods do not impact Tendermint consensus in any way and instead exist to
power subscriptions and queries of Tendermint state. 

For more information, see the [Tendermint Core ABCI methods and types specification](https://docs.tendermint.com/master/spec/abci/abci.html) 

# Event Types                       <!-- omit in toc -->

- [`nibiru.perp.v1.PositionChangedEvent`](#nibiruperpv1positionchangedevent): Event omitted when a position changes
- [`nibiru.perp.v1.PositionLiquidatedEvent`](#nibiruperpv1positionliquidatedevent): Event emitted when a position is liquidated.
- [`nibiru.perp.v1.PositionSettledEvent`](#nibiruperpv1positionsettledevent): Event emitted when a position is settled.
- [`nibiru.perp.v1.FundingRateChangedEvent`](#nibiruperpv1fundingratechangedevent):

<!-- TODO Create tx fee event -->

```ts
interface Coin {
  denom: string; // Cosmos token Bech 32 address
  amount: number; // Amount of tokens 
}
```

## `nibiru.perp.v1.PositionChangedEvent`

| Attribute (type) | Description |
| ---------------- | ----------  |
| "trader_address" (`string`) | Owner of the position |
| "pair" (`string`) | Identifier for the virtual pool corresponding to the position |
| "margin" (`Coin`) | Amount of margin backing the position |
| "position_notional" (`Dec`) |  |
| "exchanged_position_size" (`Dec`) | magnitude of the change to the position size. Recall that the position size is the number of base assets for the perp position. E.g. an ETH:USD perp with 3 ETH of exposure has a posiiton size of 3. |
| "transaction_fee" (`Coin`) | Transaction fee paid |
| "position_size" (`Dec`) | Position size (base asset value) after the change |
| "realized_pnl" (`Dec`) | Realized profits and losses after the change |
| "bad_debt" (`Coin`) | Amount of bad debt cleared by the PerpEF during the change. Bad debt is negative net margin past the liquidation point of a position. |
| "unrealized_pnl_after" (`Dec`) | Unrealized PnL after the change |
| "liquidation_penalty" (`Dec`) | Amount of margin lost due to liquidation, whether partial or full. |
| "spot_price" (`Dec`) | Spot price, synonymous with mark price in this context, is the quotient of the quote reserves and base reserves. |
| "funding_payment" (`Dec`) | A funding payment made or received by the trader on the current position. A "funding_payment" is positive if 'owner' is the sender and negative if 'owner' is the receiver of the payment. Its magnitude is abs(size * fundingRate). Funding payments act to converge the mark price and index price (average price on major spot exchanges). |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |

## `nibiru.perp.v1.PositionLiquidatedEvent`

Event emitted when a position is liquidated.
Corresponds to the proto message, `PositionLiquidatedEvent`.

| Attribute (type) | Description |
| ---------------- | ----------  |
| bad_debt (`Coin`) | Bad debt (margin units) cleared by the PerpEF during the tx. Bad debt is negative net margin past the liquidation point of a position. |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |
| exchanged_position_size (`Dec`) | TODOExchanged |
| exchanged_quote_amount (`Dec`) | TODOExchanged |
| fee_to_liquidator (`Coin`) | TODOdescr | 
| fee_to_ecosystem_fund (`Coin`) | TODOdescr |
| liquidator_address (`string`) | Address of the account that executed the tx |
| mark_price (`Dec`) | Spot price of the virtual pool after liquidation |
| margin (`Coin`) | Amount of margin remaining in the position after the liquidation |
| pair (`string`) |  Identifier for the virtual pool corresponding to the position |
| position_notional (`Dec`) | Reamining position notional (quote units) after liquiation  |
| position_size (`Dec`) | Remaing position size (base units) after liquidation |
| trader_address (`string`) | Owner of the position |
| unrealized_pnl (`Dec`) | Unrealized PnL in the position after liquidation |

## `nibiru.perp.v1.PositionSettledEvent`

| Attribute (type) | Description |
| ---------------- | ----------  |
| attribute (`type`) | TODOdescr |
| attribute (`type`) | TODOdescr |
| attribute (`type`) | TODOdescr |
| attribute (`type`) | TODOdescr |


## `nibiru.perp.v1.FundingRateChangedEvent`

| Attribute (type) | Description |
| ---------------- | ----------  |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |
| cumulative_funding_rate (`Dec`) | TODOdescr |
| index_price (`Dec`) | TODOdescr |
| latest_funding_rate (`Dec`) | TODOdescr |
| mark_price (`Dec`) | TODOdescr |
| pair (`string`) | TODOdescr |

<!--  Template for other event specs

| Attribute (type) | Description |
| ---------------- | ----------  |
| attribute (`type`) | TODOdescr |

-->
