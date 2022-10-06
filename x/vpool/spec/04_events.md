# Events

- [Event Types](#event-types)
  - [ReserveSnapshotSavedEvent](#reservesnapshotsavedevent)
  - [SwapQuoteForBaseEvent](#swapquoteforbaseevent)
  - [SwapBaseForQuoteEvent](#swapbaseforquoteevent)
  - [MarkPriceChangedEvent](#markpricechangedevent)

## Event Types

- `nibiru.vpool.v1.ReserveSnapshotSavedEvent`: Event omitted when a new reserve snapshot has been saved
- `nibiru.vpool.v1.SwapQuoteForBaseEvent`: Event emitted when quote is swapped for base.
- `nibiru.vpool.v1.SwapBaseForQuoteEvent`: Event emitted when base is swapped for quote.
- `nibiru.vpool.v1.MarkPriceChangedEvent`: Event emitted when the mark price of the pool changed.

### ReserveSnapshotSavedEvent

| Attribute (type)         | Description                        |
| ------------------------ | ---------------------------------- |
| Pair (`string`)          | The assets pair of the pool.       |
| QuoteReserve (`sdk.Dec`) | The quote reserve of the snapshot. |
| BaseReserve (`sdk.Dec`)  | The base reserve of the snapshot.  |

### SwapQuoteForBaseEvent

| Attribute (type)        | Description                  |
| ----------------------- | ---------------------------- |
| Pair (`string`)         | The assets pair of the pool. |
| QuoteAmount (`sdk.Dec`) | The quote amount swapped.    |
| BaseAmount (`sdk.Dec`)  | The base amount received.    |

### SwapBaseForQuoteEvent

| Attribute (type)        | Description                  |
| ----------------------- | ---------------------------- |
| Pair (`string`)         | The assets pair of the pool. |
| QuoteAmount (`sdk.Dec`) | The quote amount received.   |
| BaseAmount (`sdk.Dec`)  | The base amount swapped.     |

### MarkPriceChangedEvent

| Attribute (type)      | Description                  |
| --------------------- | ---------------------------- |
| Pair (`string`)       | The assets pair of the pool. |
| Price (`sdk.Dec`)     | The new price for the pool.  |
| Timestamp (`stdtime`) | The time of the event.       |
