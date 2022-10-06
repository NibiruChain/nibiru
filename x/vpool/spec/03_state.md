# State

- [Vpool](#vpool)
- [ReserveSnapshot](#reservesnapshot)

## Vpool

0x00 | Pair -> VPool

| Attribute              | Type      | Description                                                                       |
| ---------------------- | --------- | --------------------------------------------------------------------------------- |
| Pair                   | AssetPair | The market associated with the position.                                          |
| BaseAssetReserve       | sdk.Dec   | The amount of base asset in the pool.                                             |
| QuoteAssetReserve      | sdk.Dec   | The amount of quote asset in the pool.                                            |
| TradeLimitRatio        | sdk.Dec   | Ratio applied to reserves in order not to over trade.                             |
| FluctuationLimitRatio  | sdk.Dec   | Percentage that a single open/close position can alter the reserve amounts.       |
| MaxOracleSpreadRatio   | sdk.Dec   | Maximum spread allows before also considering index price to liquidate positions. |
| MaintenanceMarginRatio | sdk.Dec   | Minimum maintenance margin ratio allowed before a position can be liquidated.     |
| MaxLeverage            | sdk.Dec   | Max leverage allowed when opening a new transaction.                              |

## ReserveSnapshot

0x01 | Pair | BlockTimeMs -> ReserveSnapshot

| Attribute         | Type      | Description                                   |
| ----------------- | --------- | --------------------------------------------- |
| Pair              | AssetPair | The market pair associated with this metadata |
| BaseAssetReserve  | sdk.Dec   | The amount of base asset in the pool.         |
| QuoteAssetReserve | sdk.Dec   | The amount of quote asset in the pool.        |
| Timestamp         | int64     | The timestamp of the snapshot in milliseconds |
| BlockNumber       | int64     | The block number of the snapshot              |
