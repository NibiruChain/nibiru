# Events

- [Event Types](#event-types)
  - [PositionChangedEvent](#positionchangedevent)
  - [PositionLiquidatedEvent](#positionliquidatedevent)
  - [PositionSettledEvent](#positionsettledevent)
  - [FundingRateChangedEvent](#fundingratechangedevent)

## Event Types
- `nibiru.perp.v1.PositionChangedEvent`: Event omitted when a position changes
- `nibiru.perp.v1.PositionLiquidatedEvent`: Event emitted when a position is liquidated.
- `nibiru.perp.v1.PositionSettledEvent`: Event emitted when a position is settled.
- `nibiru.perp.v1.FundingRateChangedEvent`: Event emitted when a funding rates are calculated.

### PositionChangedEvent

| Attribute (type) | Description |
| --- | ---  |
| bad_debt (`Coin`) | Amount of bad debt cleared by the PerpEF during the change. Bad debt is negative net margin past the liquidation point of a position. |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |
| exchanged_position_size (`Dec`) | magnitude of the change to the position size. Recall that the position size is the number of base assets for the perp position. E.g. an ETH:USD perp with 3 ETH of exposure has a posiiton size of 3. |
| funding_payment (`Dec`) | A funding payment made or received by the trader on the current position. A "funding_payment" is positive if 'owner' is the sender and negative if 'owner' is the receiver of the payment. Its magnitude is abs(size * fundingRate). Funding payments act to converge the mark price and index price (average price on major spot exchanges). |
| liquidation_penalty (`Dec`) | Amount of margin lost due to liquidation, whether partial or full. |
| margin (`Coin`) | Amount of margin backing the position |
| pair (`string`) | Identifier for the virtual pool corresponding to the position. A pair is of the form `basedenom:quote`. E.g. `uatom:unusd`. |
| position_notional (`Dec`) |  |
| position_size (`Dec`) | Position size (base asset value) after the change |
| realized_pnl (`Dec`) | Realized profits and losses after the change |
| spot_price (`Dec`) | Spot price, synonymous with mark price in this context, is the quotient of the quote reserves and base reserves. |
| trader_address (`string`) | Owner of the position |
| transaction_fee (`Coin`) | Transaction fee paid |
| unrealized_pnl_after (`Dec`) | Unrealized PnL after the change |

### PositionLiquidatedEvent

Event emitted when a position is liquidated.
Corresponds to the proto message, `PositionLiquidatedEvent`.

| Attribute (type) | Description |
| ---------------- | ----------  |
| bad_debt (`Coin`) | Bad debt (margin units) cleared by the PerpEF during the tx. Bad debt is negative net margin past the liquidation point of a position. |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |
| exchanged_position_size (`Dec`) | magnitude of the change to the position size (base) |
| exchanged_quote_amount (`Dec`) | magnitude of the change to the position notional (quote) |
| fee_to_liquidator (`Coin`) | Transaction fee paid to the liquidator |
| fee_to_ecosystem_fund (`Coin`) | Transaction fee paid to the Nibi-Perps Ecosystem Fund |
| liquidator_address (`string`) | Address of the account that executed the tx |
| mark_price (`Dec`) | Spot price of the virtual pool after liquidation |
| margin (`Coin`) | Amount of margin remaining in the position after the liquidation |
| pair (`string`) | Identifier for the virtual pool corresponding to the position. A pair is of the form `basedenom:quote`. E.g. `uatom:unusd`. |
| position_notional (`Dec`) | Reamining position notional (quote units) after liquiation  |
| position_size (`Dec`) | Remaing position size (base units) after liquidation |
| trader_address (`string`) | Owner of the position |
| unrealized_pnl (`Dec`) | Unrealized PnL in the position after liquidation |

### PositionSettledEvent

| Attribute (type) | Description |
| ---------------- | ----------  |
| pair (`string`) | Identifier for the virtual pool corresponding to the position. A pair is of the form `basedenom:quote`. E.g. `uatom:unusd`. |
| settled_coins (`[]Coin`) | Coins transferred during the settlement |
| trader_address (`string`) | Owner of the position |

### FundingRateChangedEvent

| Attribute (type) | Description |
| ---------------- | ----------  |
| block_height (`int64`) | Block number at which the position changed |
| block_time_ms (`int64`) | Block time in Unix milliseconds at which the position changed. |
| cumulative_funding_rate (`Dec`) | Cumulative funding rate. The sum of the cumulative funding rates (CFPs) for the pair. The funding payment paid by a user is the `(latestCPF - lastUpdateCPF) * positionSize`, where `lastUpdateCPF` is the last cumulative funding payment the position applied and `latestCPF` is the most recent CPF for the virtual pool. |
| index_price (`Dec`) | Price of the "underlying" for the perpetual swap. |
| latest_funding_rate (`Dec`) | Most recent value for the funding rate.  |
| mark_price (`Dec`) | Instantaneous derivate price for the perp position. Equivalent to the quotient of the quote and base reserves. |
| pair (`string`) | Identifier for the virtual pool corresponding to the position. A pair is of the form `basedenom:quote`. E.g. `uatom:unusd`. |

<!--  Template for other event specs

| Attribute (type)   | Description     |
| ------------------ | --------------- |
| attribute (`type`) | TODOdescription |

-->
