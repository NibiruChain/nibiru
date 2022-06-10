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

- [`transfer`](#transfer): Emitted when assets are transferred between addresses as a result of a Nibiru protocol.
- [`position_change`](#position_change): Emitted when a position state changes. 
- [`position_liquidate`](#position_liquidate): 
- [`position_settle`](#position_settle): 
- [`margin_ratio_change`](#margin_ratio_change): 
- [`margin_change`](#margin_change): Emitted when the margin of a single  
- [`internal_position_response`](#internal_position_response): 

<!-- TODO Create tx fee event -->

## `transfer`

Attributes - "transfer:

- "from": Sender address 
- "to": Receiver address 
- "denom": Cosmos-SDK token Bech 32 address for the token
- "amount": Amount of tokens sent in the transfer


##  `position_change`

Attributes - "position_change":

- "owner": Address 
- "vpool": 
- "margin": 
- "notional": 
- "vsizeChange": magnitude of the change to the position size. Recall that the position size is the number of base assets for the perp position. E.g. an ETH:USD perp with 3 ETH of exposure has a posiiton size of 3.
- "txFee": 
- "vsizeAfter": 
- "realizedPnlAfter": 
- "bad_debt": 
- "unrealized_pnl_after": 
- "liquidation_penalty": 
- "mark_price": 
- "funding_payment": A funding payment made or received by the trader on the current position. A "funding_payment" is positive if 'owner' is the sender and negative if 'owner' is the receiver of the payment. Its magnitude is abs(size * fundingRate). Funding payments act to converge the mark price and index price (average price on major spot exchanges).

## `position_liquidate`


## `position_settle`

## `margin_ratio_change`

## `margin_change`

## `internal_position_response`