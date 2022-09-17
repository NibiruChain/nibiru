# State

## Position

|Attribute|Type|Description|
|---|---|---|
|TraderAddress|string|The trader's bech32 address.|
|Pair|AssetPair|The market associated with the position.|
|Size|sdk.Dec|The position size.|
|Margin|sdk.Dec|The amount of collateral backing the position.|
|OpenNotional|sdk.Dec|The notional value from when the position was opened. Used to calculate PnL.|
|LatestCumulativeFundingRate|sdk.Dec|The last funding rate applied on the position.|
|BlockNumber|int64|The last block number this position was updated.|

## PairMetadata

|Attribute|Type|Description|
|---|---|---|
|Pair|AssetPair|The market pair associated with this metadata|
|CumulativeFundingRates|[]sdk.Dec|Historial list of funding rates for a given market. Calculated once per epoch.|

## PrepaidBadDebt

|Attribute|Type|Description|
|---|---|---|
|denom|string|The denomination of the prepaid bad debt.|
|amount|sdk.Int|The amount of prepaid bad debt.|
